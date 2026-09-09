package cli

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"myenv/internal/backend"
	"myenv/internal/config"
	"myenv/internal/runner"
	"path"
)

func TestPythonMirrorsRealCLI(t *testing.T) {
	testPythonMirrorsRealCLI(t, false, false)
}

func TestPythonMirrorsCustomCARealCLI(t *testing.T) {
	testPythonMirrorsRealCLI(t, true, false)
}

func TestPowerShellAppliedProfile(t *testing.T) {
	testPythonMirrorsRealCLI(t, false, true)
}

func TestPOSIXAppliedProfile(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux shell integration")
	}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			if _, err := exec.LookPath(shell); err != nil {
				t.Skipf("requires %s", shell)
			}
			testPythonMirrorsRealCLI(t, false, true, shell)
		})
	}
}

// The generated wrapper targets this test executable. Dispatch only its run
// invocation into the real CLI with an isolated user configuration namespace.
func init() {
	if namespace := os.Getenv("MYENV_TEST_SHELL_NAMESPACE"); namespace != "" && len(os.Args) > 1 && os.Args[1] == "run" {
		os.Exit(execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, "test", namespace))
	}
}

func testPythonMirrorsRealCLI(t *testing.T, secure, profile bool, shells ...string) {
	shell := "powershell"
	if len(shells) != 0 {
		shell = shells[0]
	}
	uvArchive, pythonArchive := os.Getenv("MYENV_TEST_UV_ARCHIVE"), os.Getenv("MYENV_TEST_PYTHON_ARCHIVE")
	if uvArchive == "" || pythonArchive == "" {
		t.Skip("requires retained native uv and CPython 3.12.13 archives")
	}
	if profile && shell == "powershell" && runtime.GOOS != "windows" {
		t.Skip("PowerShell profile integration runs on Windows")
	}
	platform, err := runner.Platform()
	if err != nil {
		t.Fatal(err)
	}
	release, err := backend.FixedUVRelease(platform)
	if err != nil {
		t.Fatal(err)
	}
	var uvDownloads, pythonDownloads atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/uv/"+backend.UVVersion+"/"+path.Base(release.URL):
			uvDownloads.Add(1)
			http.ServeFile(w, r, uvArchive)
		case strings.HasPrefix(r.URL.Path, "/python/") && strings.Contains(r.URL.Path, "cpython-3.12.13"):
			pythonDownloads.Add(1)
			http.ServeFile(w, r, pythonArchive)
		default:
			http.NotFound(w, r)
		}
	}))
	if secure {
		certificate, rootPEM := mirrorCertificate(t)
		server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
		server.StartTLS()
		certificateFile := filepath.Join(t.TempDir(), "ca.pem")
		if err := os.WriteFile(certificateFile, rootPEM, 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SSL_CERT_FILE", certificateFile)
		t.Setenv("SSL_CERT_DIR", "")
	} else {
		server.Start()
	}
	defer server.Close()
	t.Setenv("MYENV_UV_MIRROR", server.URL+"/uv")
	t.Setenv("MYENV_PYTHON_MIRROR", server.URL+"/python")
	t.Setenv("MYENV_NODE_MIRROR", "")
	root, namespace := t.TempDir(), t.TempDir()
	declaration := filepath.Join(root, "myenv.yaml")
	if profile {
		var err error
		declaration, err = config.ProfilePath(namespace)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(declaration), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(declaration, []byte("schema: 1\ntools: {python: '3.12.13'}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	invoke := func(args ...string) {
		t.Helper()
		out.Reset()
		diagnostic.Reset()
		if profile && args[0] != "shell-init" {
			args = append([]string{args[0], "--global"}, args[1:]...)
		}
		if code := execute(append([]string{"-C", root}, args...), bytes.NewReader(nil), &out, &diagnostic, "test", namespace); code != 0 {
			t.Fatalf("%v: %d %s %s", args, code, out.String(), diagnostic.String())
		}
	}
	invoke("sync", "--no-input", "--json")
	var applied struct {
		OK, Changed bool
		Data        struct{ Generation struct{ ID string } }
	}
	if err := json.Unmarshal(out.Bytes(), &applied); err != nil || !applied.OK || !applied.Changed || applied.Data.Generation.ID == "" {
		t.Fatalf("initial sync: %s %v", out.String(), err)
	}
	if uvDownloads.Load() != 1 || pythonDownloads.Load() != 1 {
		t.Fatalf("downloads: uv=%d python=%d", uvDownloads.Load(), pythonDownloads.Load())
	}
	server.Close()
	invoke("sync", "--locked", "--no-input", "--json")
	var unchanged struct {
		OK, Changed bool
		Data        struct{ Generation struct{ ID string } }
	}
	if err := json.Unmarshal(out.Bytes(), &unchanged); err != nil || !unchanged.OK || unchanged.Changed || unchanged.Data.Generation.ID != applied.Data.Generation.ID {
		t.Fatalf("offline sync: %s %v", out.String(), err)
	}
	invoke("run", "python", "--version")
	if strings.TrimSpace(out.String()) != "Python 3.12.13" {
		t.Fatalf("installed Python: %s", out.String())
	}
	if profile {
		invoke("shell-init", shell)
		shellEnvironment := append(os.Environ(), "MYENV_TEST_SHELL_NAMESPACE="+namespace)
		if executable := os.Getenv("MYENV_TEST_SHELL_CLI_EXECUTABLE"); executable != "" {
			if !filepath.IsAbs(executable) {
				t.Fatal("shell CLI artifact must be absolute")
			}
			configVariable := "APPDATA="
			if runtime.GOOS == "linux" {
				configVariable = "XDG_CONFIG_HOME="
			}
			shellEnvironment = append(os.Environ(), configVariable+namespace, "MYENV_TEST_SHELL_NAMESPACE=")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			command := exec.CommandContext(ctx, executable, "shell-init", shell)
			command.Env = shellEnvironment
			generated, err := command.Output()
			cancel()
			if err != nil {
				t.Fatalf("artifact shell-init: %v", err)
			}
			out.Reset()
			out.Write(generated)
			t.Logf("shell-init and run use artifact %s", executable)
		}
		driver := filepath.Join(root, "profile-shell.ps1")
		script := out.String() + "python -I -c 'import sys; print(sys.version.split()[0]); sys.exit(23)'\nexit $LASTEXITCODE\n"
		shellExecutable := "pwsh"
		options := []string{"-NoProfile", "-NonInteractive", "-File", driver}
		if shell != "powershell" {
			shellExecutable = shell
			options = []string{"-f", driver}
			script = strings.ReplaceAll(script, "$LASTEXITCODE", "$?")
			if shell == "bash" {
				options = []string{"--noprofile", "--norc", driver}
			}
			if shell == "fish" {
				options = []string{"--no-config", "--private", driver}
				script = strings.ReplaceAll(script, "$?", "$status")
			}
		}
		if err := os.WriteFile(driver, []byte(script), 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, shellExecutable, options...)
		command.Env = shellEnvironment
		output, err := command.CombinedOutput()
		if command.ProcessState == nil || command.ProcessState.ExitCode() != 23 || strings.TrimSpace(string(output)) != "3.12.13" {
			t.Fatalf("applied profile wrapper: %v %s", err, output)
		}
	}
}

func mirrorCertificate(t *testing.T) (tls.Certificate, []byte) {
	t.Helper()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	rootDER, err := x509.CreateCertificate(rand.Reader, root, root, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err = x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), NotBefore: root.NotBefore, NotAfter: root.NotAfter, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: leafKey}, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER})
}

package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestLinuxTerminalInputAndInterrupt(t *testing.T) {
	testLinuxTerminalSignal(t, "SIGINT", "03")
}

func TestLinuxTerminalInputAndQuit(t *testing.T) {
	testLinuxTerminalSignal(t, "SIGQUIT", "1c")
}

func TestLinuxTerminalStopAndContinue(t *testing.T) {
	testLinuxTerminalSignal(t, "SIGINT", "03")
}

func TestLinuxTerminalStopBySignal(t *testing.T) {
	testLinuxTerminalSignal(t, "SIGINT", "03")
}

func TestLinuxTerminalStopOwnerKilled(t *testing.T) {
	testLinuxTerminalSignal(t, "SIGINT", "03")
}

func TestLinuxTerminalStopPendingTerm(t *testing.T) {
	testLinuxTerminalSignal(t, "SIGTERM", "")
}

func TestLinuxShellForegroundResume(t *testing.T) {
	testLinuxTerminalSignal(t, "SIGINT", "03")
}

func TestLinuxShellBackgroundResume(t *testing.T) {
	testLinuxTerminalSignal(t, "SIGINT", "03")
}

func TestLinuxShellRunningBackground(t *testing.T) {
	testLinuxTerminalSignal(t, "SIGINT", "03")
}

func testLinuxTerminalSignal(t *testing.T, signalName, controlHex string) {
	record := os.Getenv("MYENV_TEST_PREPARED_RECORD")
	if record == "" {
		t.Skip("requires retained Linux Node and Python PTY fixture")
	}
	if os.Getenv("MYENV_TERMINAL_HELPER") == "1" {
		data, err := os.ReadFile(record)
		if err != nil {
			t.Fatal(err)
		}
		var prepared struct{ Executable string }
		if err = json.Unmarshal(data, &prepared); err != nil {
			t.Fatal(err)
		}
		script := "let count=0; process.on('SIGINT',()=>{if(++count===1)setTimeout(()=>process.exit(count===1?23:90+count),100)}); process.stdin.setEncoding('utf8'); process.stdin.once('data',s=>{if(s!=='hello\\n')process.exit(81);process.stdout.write('input-ok\\n')});process.stdout.write('ready\\n');setInterval(()=>{},1000)"
		script = strings.ReplaceAll(script, "SIGINT", signalName)
		if strings.HasPrefix(t.Name(), "TestLinuxTerminalStop") || strings.HasPrefix(t.Name(), "TestLinuxShell") {
			script += ";let resumes=0;process.on('SIGCONT',()=>{const n=++resumes;process.stdin.removeAllListeners('data');process.stdin.once('data',s=>{if(s!=='resume-'+n+'\\n')process.exit(82);process.stdout.write('resumed-input-'+n+'\\n')});process.stdout.write('continued-'+n+'\\n')})"
		}
		if t.Name() == "TestLinuxShellRunningBackground" {
			script = strings.ReplaceAll(script, "const n=++resumes;", "const n=++resumes;if(n===1){process.stdin.pause();process.stdout.write('background-running\\n');return;}")
			script += ";process.stdout.write('node-pid:'+process.pid+'\\n')"
		}
		if strings.HasPrefix(t.Name(), "TestLinuxTerminalStop") {
			script += ";process.stdout.write('node-pid:'+process.pid+'\\n')"
		}
		code, err := Execute(context.Background(), Process{Executable: prepared.Executable, Args: []string{"-e", script}, Environment: os.Environ(), Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr})
		if err != nil || code != 23 {
			t.Fatalf("terminal child: %d %v", code, err)
		}
		assertForeground := func() {
			foreground, err := unix.IoctlGetInt(0, unix.TIOCGPGRP)
			if err != nil || foreground != syscall.Getpgrp() {
				t.Fatalf("foreground not restored: %d %v", foreground, err)
			}
		}
		assertForeground()
		// Check the actual exec boundary before a runtime can reuse these slots
		// for its own descriptors. Shell test uses builtins only.
		code, err = Execute(context.Background(), Process{
			Executable: "/bin/sh",
			Args:       []string{"-c", "for fd in 3 4 5 6 7 8; do if test -e /proc/self/fd/$fd; then exit 86; fi; done"},
			Stdin:      os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr,
		})
		if err != nil || code != 0 {
			t.Fatalf("supervisor descriptor inherited by terminal child: %d %v", code, err)
		}
		assertForeground()
		_, err = Execute(context.Background(), Process{Executable: os.Args[0] + ".missing-terminal-fixture", Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr})
		if err == nil {
			t.Fatal("missing executable unexpectedly started")
		}
		assertForeground()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		code, err = Execute(ctx, Process{Executable: "/bin/sh", Args: []string{"-c", "sleep 5"}, Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr})
		cancel()
		if err != nil || code == 0 {
			t.Fatalf("terminal cancellation: %d %v", code, err)
		}
		assertForeground()
		foreground, err := unix.IoctlGetInt(0, unix.TIOCGPGRP)
		if err != nil || foreground != syscall.Getpgrp() {
			t.Fatalf("foreground not restored: %d %v", foreground, err)
		}
		fmt.Println("caller-ready")
		data = make([]byte, 6)
		if _, err = io.ReadFull(os.Stdin, data); err != nil || string(data) != "again\n" {
			t.Fatal("caller input", string(data), err)
		}
		fmt.Println("restored")
		os.Exit(23)
	}
	if _, err := os.Stat("/usr/bin/python3"); err != nil {
		t.Fatal("Python PTY fixture unavailable", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "/usr/bin/python3", "-c", terminalPTYFixture, os.Args[0], t.Name(), controlHex)
	command.Env = append(os.Environ(), "MYENV_TERMINAL_HELPER=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("PTY validation: %v\n%s", err, output)
	}
}

const terminalPTYFixture = `
import errno, os, pty, select, signal, sys, time, shlex, re
shell = sys.argv[2].startswith('TestLinuxShell')
background = sys.argv[2] == 'TestLinuxShellBackgroundResume'
running_background = sys.argv[2] == 'TestLinuxShellRunningBackground'
pending_term = sys.argv[2] == 'TestLinuxTerminalStopPendingTerm'
pid, master = pty.fork()
if pid == 0:
    if shell:
        os.environ['PS1'] = 'myenv-test-prompt> '
        os.environ['HISTFILE'] = '/dev/null'
        os.execv('/bin/bash', ['bash', '--noprofile', '--norc', '-i'])
    os.execv(sys.argv[1], [sys.argv[1], '-test.run=^' + sys.argv[2] + '$'])
waited = False
transcript = b''
def expect(marker):
    global transcript
    deadline = time.monotonic() + 4
    while marker not in transcript:
        if time.monotonic() >= deadline:
            raise RuntimeError('PTY timeout: ' + repr(transcript))
        if select.select([master], [], [], 0.1)[0]:
            try:
                block = os.read(master, 4096)
            except OSError as e:
                raise RuntimeError('PTY ended: ' + repr(transcript)) from e
            if not block or len(transcript) + len(block) > 65536:
                raise RuntimeError('invalid PTY output: ' + repr(transcript))
            transcript += block
try:
    if shell:
        expect(b'myenv-test-prompt> ')
        transcript = b''
        command = shlex.quote(sys.argv[1]) + ' ' + shlex.quote('-test.run=^' + sys.argv[2] + '$')
        os.write(master, (command + '\n').encode())
    expect(b'ready\r\n')
    os.write(master, b'hello\n')
    expect(b'input-ok\r\n')
    if running_background or sys.argv[2].startswith('TestLinuxTerminalStop'):
        node_pid = int(re.search(rb'node-pid:(\d+)\r\n', transcript).group(1))
    if shell or sys.argv[2].startswith('TestLinuxTerminalStop'):
        for cycle in (1, 2):
            transcript = b''
            if sys.argv[2] == 'TestLinuxTerminalStopBySignal':
                os.kill(pid, signal.SIGTSTP)
            else:
                os.write(master, b'\x1a')
            deadline = time.monotonic() + 4
            while not shell:
                observed, status = os.waitpid(pid, os.WNOHANG | os.WUNTRACED)
                if observed:
                    if not os.WIFSTOPPED(status):
                        raise RuntimeError('caller exited instead of stopping')
                    break
                if time.monotonic() >= deadline:
                    raise RuntimeError('caller did not stop after Ctrl-Z')
                time.sleep(0.01)
            if shell:
                expect(b'Stopped')
                expect(b'myenv-test-prompt> ')
                transcript = b''
                if (background or running_background) and cycle == 1:
                    os.write(master, b'bg; printf "bg-returned\\n"\n')
                    expect(b'bg-returned\r\n')
                    expect(b'myenv-test-prompt> ')
                    if os.tcgetpgrp(master) != pid:
                        raise RuntimeError('background task stole terminal foreground')
                    if running_background:
                        expect(b'background-running\r\n')
                        transcript = b''
                        os.write(master, b'jobs; printf "running-checked\\n"\n')
                        expect(b'running-checked\r\n')
                        if b'Running' not in transcript or b'Stopped' in transcript:
                            raise RuntimeError('fixture was not a running background job: ' + repr(transcript))
                        transcript = b''
                        os.write(master, b'fg; printf "job-exit:%s\\n" "$?"\n')
                        deadline = time.monotonic() + 4
                        while os.tcgetpgrp(master) != node_pid:
                            if time.monotonic() >= deadline:
                                raise RuntimeError('fg did not restore running child foreground')
                            time.sleep(0.01)
                        break
                    # The user's pending terminal read should stop the job.
                    deadline = time.monotonic() + 4
                    while True:
                        transcript = b''
                        os.write(master, b'jobs; printf "jobs-checked\\n"\n')
                        expect(b'jobs-checked\r\n')
                        if b'Stopped' in transcript:
                            break
                        if time.monotonic() >= deadline:
                            raise RuntimeError('background terminal reader did not stop')
                        time.sleep(0.02)
                    transcript = b''
                os.write(master, b'fg; printf "job-exit:%s\\n" "$?"\n')
            else:
                with open('/proc/%d/stat' % node_pid) as stat_file:
                    state = stat_file.read().rsplit(')', 1)[1].split()[0]
                if state != 'T':
                    raise RuntimeError('caller stopped without stopping child')
                if sys.argv[2] == 'TestLinuxTerminalStopOwnerKilled':
                    os.kill(pid, signal.SIGKILL)
                    _, status = os.waitpid(pid, 0)
                    waited = True
                    if not os.WIFSIGNALED(status) or os.WTERMSIG(status) != signal.SIGKILL:
                        raise RuntimeError('caller was not killed')
                    deadline = time.monotonic() + 4
                    while os.path.exists('/proc/%d' % node_pid):
                        if time.monotonic() >= deadline:
                            raise RuntimeError('stopped child survived caller death')
                        time.sleep(0.01)
                    sys.exit(0)
                if pending_term:
                    os.kill(pid, signal.SIGTERM)
                    with open('/proc/%d/status' % pid) as status_file:
                        fields = dict(line.split(':', 1) for line in status_file if ':' in line)
                    pending = int(fields['SigPnd'].strip(), 16) | int(fields['ShdPnd'].strip(), 16)
                    if not pending & (1 << (signal.SIGTERM - 1)):
                        raise RuntimeError('TERM was not pending on stopped caller')
                os.kill(pid, signal.SIGCONT)
                if pending_term:
                    break
            resumed = cycle + (1 if background else 0)
            expect(('continued-%d\r\n' % resumed).encode())
            os.write(master, ('resume-%d\n' % resumed).encode())
            expect(('resumed-input-%d\r\n' % resumed).encode())
    if not pending_term:
        os.write(master, bytes.fromhex(sys.argv[3]))
    expect(b'caller-ready\r\n')
    os.write(master, b'again\n')
    expect(b'restored\r\n')
    if shell:
        expect(b'job-exit:23\r\n')
        os.write(master, b'exit 0\n')
    _, status = os.waitpid(pid, 0)
    waited = True
    if os.waitstatus_to_exitcode(status) != (0 if shell else 23):
        raise RuntimeError('wrong caller status: ' + repr(transcript))
finally:
    if not waited:
        try: os.kill(pid, signal.SIGKILL)
        except ProcessLookupError: pass
        os.waitpid(pid, 0)
    os.close(master)
`

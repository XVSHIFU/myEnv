package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
	"myenv/internal/config"
)

func terminalInput(in io.Reader, out io.Writer) bool {
	if os.Getenv("CI") != "" {
		return false
	}
	input, ok := in.(*os.File)
	if !ok {
		return false
	}
	output, ok := out.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(input.Fd())) && term.IsTerminal(int(output.Fd()))
}

func buildPrompt(in io.Reader, out io.Writer, languages ...locale) func(string) (bool, error) {
	var lang locale
	if len(languages) > 0 {
		lang = languages[0]
	}
	return func(reason string) (bool, error) {
		if _, err := fmt.Fprintf(out, lang.text("%s\nAllow dependency and project build code to run as your user for this invocation? [y/N]: "), lang.text(reason)); err != nil {
			return false, err
		}
		scanner := bufio.NewScanner(in)
		scanner.Buffer(make([]byte, 256), 4096)
		if !scanner.Scan() {
			return false, scanner.Err()
		}
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return answer == "y" || answer == "yes", nil
	}
}

func versionPrompt(in io.Reader, out io.Writer, languages ...locale) config.SelectVersion {
	var lang locale
	if len(languages) > 0 {
		lang = languages[0]
	}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024), 4096)
	return func(tool, reason string) (string, string, error) {
		if _, err := fmt.Fprintln(out, lang.text(reason)); err != nil {
			return "", "", err
		}
		prompt := lang.text("Enter tool@version (python, node, java, go, rust): ")
		if tool != "" {
			prompt = fmt.Sprintf(lang.text("Enter %s version: "), tool)
		}
		if _, err := fmt.Fprint(out, prompt); err != nil {
			return "", "", err
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", "", err
			}
			return "", "", &config.NeedsInput{Message: "input ended; supply a version declaration and retry"}
		}
		value := strings.TrimSpace(scanner.Text())
		if tool == "" {
			var found bool
			tool, value, found = strings.Cut(value, "@")
			if !found {
				return "", "", &config.NeedsInput{Message: "enter a supported tool@<version>"}
			}
		}
		if value == "" {
			return "", "", &config.NeedsInput{Message: "no version selected; supply a version declaration and retry"}
		}
		return tool, value, nil
	}
}

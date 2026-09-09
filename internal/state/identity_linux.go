package state

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func processIdentity(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid process ID")
	}
	read := func(path string) ([]byte, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, 8193))
		if len(data) > 8192 {
			return nil, fmt.Errorf("process identity input too large")
		}
		return data, err
	}
	stat, err := read(fmt.Sprintf("/proc/%d/stat", pid))
	if os.IsNotExist(err) {
		return "", ErrProcessGone
	}
	if err != nil {
		return "", err
	}
	// comm is parenthesized and may contain spaces or closing parentheses.
	end := strings.LastIndexByte(string(stat), ')')
	if end < 0 {
		return "", fmt.Errorf("invalid process stat")
	}
	fields := strings.Fields(string(stat[end+1:]))
	if len(fields) < 20 {
		return "", fmt.Errorf("incomplete process stat")
	}
	start, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return "", err
	}
	boot, err := read("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", err
	}
	bootID := strings.TrimSpace(string(boot))
	if bootID == "" {
		return "", fmt.Errorf("missing boot identity")
	}
	return fmt.Sprintf("linux:%s:%d", bootID, start), nil
}

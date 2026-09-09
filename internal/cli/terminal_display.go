package cli

import (
	"fmt"
	"golang.org/x/term"
	"io"
	"os"
	"sync"
	"time"
)

func terminalANSI(out io.Writer) bool {
	if p, ok := out.(*progressOutput); ok {
		out = p.out
	}
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := out.(*os.File)
	return ok && term.IsTerminal(int(f.Fd())) && enableTerminalANSI(f)
}
func paint(out io.Writer, style, text string) string {
	if terminalANSI(out) {
		return "\x1b[" + style + "m" + text + "\x1b[0m"
	}
	return text
}
func stateColor(state string) string {
	if state == "available" {
		return "32"
	}
	if state == "broken" {
		return "31"
	}
	return "33"
}

// One display per CLI invocation. Ordinary output clears animation before writing.
type progressDisplay struct {
	mu      sync.Mutex
	out     io.Writer
	message string
	frame   int
	stop    chan struct{}
	done    chan struct{}
}

func newProgressDisplay(out io.Writer) *progressDisplay { return &progressDisplay{out: out} }
func (p *progressDisplay) clearLocked() {
	if p.message != "" {
		fmt.Fprint(p.out, "\r\x1b[2K")
		p.message = ""
	}
}
func (p *progressDisplay) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clearLocked()
	return p.out.Write(b)
}
func (p *progressDisplay) phase(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.message = message
	p.renderLocked()
	if p.stop != nil {
		return
	}
	p.stop = make(chan struct{})
	p.done = make(chan struct{})
	go func() {
		defer close(p.done)
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-ticker.C:
				p.mu.Lock()
				if p.message != "" {
					p.renderLocked()
				}
				p.mu.Unlock()
			}
		}
	}()
}
func (p *progressDisplay) renderLocked() {
	fmt.Fprintf(p.out, "\r\x1b[2K\x1b[36m%c\x1b[0m %s", "|/-\\"[p.frame%4], p.message)
	p.frame++
}
func (p *progressDisplay) close() {
	if p.stop != nil {
		close(p.stop)
		<-p.done
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clearLocked()
}

type progressOutput struct {
	display *progressDisplay
	out     io.Writer
}

func (p *progressOutput) Write(b []byte) (int, error) {
	p.display.mu.Lock()
	defer p.display.mu.Unlock()
	p.display.clearLocked()
	return p.out.Write(b)
}

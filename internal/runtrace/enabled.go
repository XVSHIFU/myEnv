//go:build linux && runtrace

package runtrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

var events struct {
	sync.Mutex
	Values []event
}

type event struct {
	Name   string
	UnixNS int64
}

func Mark(name string) {
	e := event{name, time.Now().UnixNano()}
	events.Lock()
	if len(events.Values) < 256 {
		events.Values = append(events.Values, e)
	}
	events.Unlock()
}

// The dedicated diagnostic build writes once at exit, outside the timed stages.
// Each process has its own file. No user argv or environment values are recorded.
func Flush() {
	directory := os.Getenv("MYENV_RUN_TRACE_DIRECTORY")
	if !filepath.IsAbs(directory) {
		return
	}
	events.Lock()
	defer events.Unlock()
	f, err := os.OpenFile(filepath.Join(directory, strconv.Itoa(os.Getpid())+".json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(events.Values)
}

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type UIDirectory struct {
	Path string `json:"path"`
	Key  string `json:"key"`
}

func NormalizeUIDirectory(path string) (UIDirectory, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return UIDirectory{}, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return UIDirectory{}, err
	}
	if !info.IsDir() {
		return UIDirectory{}, fmt.Errorf("请选择目录")
	}
	// Windows may allow this directory while denying traversal metadata on a
	// parent. Do not reject an accessible project merely to resolve an alias.
	if resolved, e := filepath.EvalSymlinks(absolute); e == nil {
		absolute = resolved
	}
	absolute = filepath.Clean(absolute)
	key := absolute
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return UIDirectory{absolute, key}, nil
}
func (c *UIController) History() []UITask {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]UITask, len(c.tasks))
	copy(result, c.tasks)
	for i := range result {
		result[i] = cloneUITask(result[i])
	}
	return result
}

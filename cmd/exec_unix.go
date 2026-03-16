//go:build !windows

package cmd

import (
	"os"
	"syscall"
)

// replaceProcess replaces the current process with the given command (Unix exec).
// PyCharm and other IDEs see the final process directly.
func replaceProcess(path string, args []string) error {
	return syscall.Exec(path, args, os.Environ())
}

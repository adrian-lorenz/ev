//go:build windows

package cmd

import (
	"os"
	"os/exec"
)

// replaceProcess runs the command as a child process on Windows (no syscall.Exec).
func replaceProcess(path string, args []string) error {
	c := exec.Command(path, args[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

package cmd

import (
	"os/exec"
	"syscall"
)

func syscallExec(path string, args []string, env []string) error {
	return syscall.Exec(path, args, env)
}

func findExecutable(name string) (string, error) {
	return exec.LookPath(name)
}

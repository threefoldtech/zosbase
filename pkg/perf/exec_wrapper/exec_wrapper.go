package execwrapper

import (
	"context"
	"os/exec"
)

// ExecWrapper wraps the exec package functions to allow for mocking
type ExecWrapper interface {
	LookPath(file string) (string, error)
	CommandContext(ctx context.Context, name string, arg ...string) ExecCmd
}

// ExecCmd represents a command that can be executed
type ExecCmd interface {
	CombinedOutput() ([]byte, error)
}

// RealExecWrapper implements ExecWrapper using the real exec package
type RealExecWrapper struct{}

// LookPath searches for an executable named file in the directories named by the PATH environment variable
func (r *RealExecWrapper) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// CommandContext returns the Cmd struct to execute the named program with the given arguments
func (r *RealExecWrapper) CommandContext(ctx context.Context, name string, arg ...string) ExecCmd {
	return &RealExecCmd{cmd: exec.CommandContext(ctx, name, arg...)}
}

// RealExecCmd wraps exec.Cmd to implement ExecCmd interface
type RealExecCmd struct {
	cmd *exec.Cmd
}

// CombinedOutput runs the command and returns its combined standard output and standard error
func (r *RealExecCmd) CombinedOutput() ([]byte, error) {
	return r.cmd.CombinedOutput()
}

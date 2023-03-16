package rcebot

import (
	"bytes"
	"os/exec"
)

// Executor handles execution of system commands.
type Executor struct {
	buffer bytes.Buffer
}

// Execute executes the given command and returns the combined output of stdout and stderr.
// The returned string is only valid until the next call to Execute.
func (e *Executor) Execute(cmd exec.Cmd) ([]byte, error) {
	cmd.Stdout = &e.buffer
	cmd.Stderr = &e.buffer
	err := cmd.Run()
	b := e.buffer.Bytes()
	e.buffer.Reset()
	return b, err
}

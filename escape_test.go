package rcebot

import (
	"errors"
	"testing"
)

const (
	outputEscape   = "`\\\\localhost`\n"
	responseEscape = "```\n\\`\\\\\\\\localhost\\`\n```\n"

	outputNoEscape   = "localhost\n"
	responseNoEscape = "```\nlocalhost\n```\n"

	outputNoLF   = "localhost"
	responseNoLF = responseNoEscape

	outputMultiLine   = "localhost\nlocalhost\n"
	responseMultiLine = "```\nlocalhost\nlocalhost\n```\n"

	errString         = "1 + 1 = 2"
	responseEscapeErr = responseEscape + "1 \\+ 1 \\= 2"
)

var testErr = errors.New(errString)

func testCommandOutputResponseBuilder(t *testing.T, rb *CommandOutputResponseBuilder, output []byte, err error, expectedResponse string) {
	t.Helper()
	response := rb.Build(output, err)
	if response != expectedResponse {
		t.Errorf("expected response %q, got %q", expectedResponse, response)
	}
}

func TestCommandOutputResponseBuilder(t *testing.T) {
	rb := CommandOutputResponseBuilder{}
	t.Run("Escape", func(t *testing.T) {
		testCommandOutputResponseBuilder(t, &rb, []byte(outputEscape), nil, responseEscape)
	})
	t.Run("NoEscape", func(t *testing.T) {
		testCommandOutputResponseBuilder(t, &rb, []byte(outputNoEscape), nil, responseNoEscape)
	})
	t.Run("NoLF", func(t *testing.T) {
		testCommandOutputResponseBuilder(t, &rb, []byte(outputNoLF), nil, responseNoLF)
	})
	t.Run("MultiLine", func(t *testing.T) {
		testCommandOutputResponseBuilder(t, &rb, []byte(outputMultiLine), nil, responseMultiLine)
	})
	t.Run("EscapeErr", func(t *testing.T) {
		testCommandOutputResponseBuilder(t, &rb, []byte(outputEscape), testErr, responseEscapeErr)
	})
}

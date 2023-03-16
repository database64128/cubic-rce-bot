package rcebot

import (
	"bytes"
	"regexp"
	"unsafe"
)

var (
	markdownV2PlaintextEscapeRegexp = regexp.MustCompile("[`\\\\_*[\\]()~>#+\\-=|{}.!]")
	markdownV2CodeBlockEscapeRegexp = regexp.MustCompile("[`\\\\]")
)

// EscapeMarkdownV2Plaintext escapes the given string for use in a MarkdownV2 plaintext.
func EscapeMarkdownV2Plaintext(s string) string {
	return markdownV2PlaintextEscapeRegexp.ReplaceAllString(s, "\\$0")
}

// EscapeMarkdownV2CodeBlock escapes the given string for use in a MarkdownV2 code block.
func EscapeMarkdownV2CodeBlock(s string) string {
	return markdownV2CodeBlockEscapeRegexp.ReplaceAllString(s, "\\$0")
}

// CommandOutputResponseBuilder is a reusable builder for command output responses.
type CommandOutputResponseBuilder struct {
	buffer        bytes.Buffer
	escapeIndexes []int
}

// Build uses the builder's internal buffer to build a response for the given command output.
// The returned string is only valid until the next call to Build.
func (rb *CommandOutputResponseBuilder) Build(output []byte, err error) string {
	var escapeCount int
	for i := range output {
		switch output[i] {
		case '`', '\\':
			rb.escapeIndexes = append(rb.escapeIndexes, i)
			escapeCount++
		}
	}

	var lfCount int
	if len(output) == 0 || output[len(output)-1] != '\n' {
		lfCount = 1
	}

	var escapedErrString string
	if err != nil {
		escapedErrString = EscapeMarkdownV2Plaintext(err.Error())
	}

	// ```\n + escaped output + \n + ```\n + escaped error string
	rb.buffer.Grow(4 + len(output) + escapeCount + lfCount + 4 + len(escapedErrString))
	rb.buffer.WriteString("```\n")

	var prevIndex int
	for _, index := range rb.escapeIndexes {
		rb.buffer.Write(output[prevIndex:index])
		rb.buffer.WriteByte('\\')
		prevIndex = index
	}
	rb.buffer.Write(output[prevIndex:])

	if lfCount != 0 {
		rb.buffer.WriteByte('\n')
	}

	rb.buffer.WriteString("```\n")
	rb.buffer.WriteString(escapedErrString)

	b := rb.buffer.Bytes()
	s := unsafe.String(unsafe.SliceData(b), len(b))
	rb.buffer.Reset()
	rb.escapeIndexes = rb.escapeIndexes[:0]
	return s
}

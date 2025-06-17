package rcebot_test

import (
	"errors"
	"testing"

	rcebot "github.com/database64128/cubic-rce-bot"
)

func TestCommandOutputResponseBuilder(t *testing.T) {
	rb := rcebot.CommandOutputResponseBuilder{}

	for _, c := range [...]struct {
		name   string
		output string
		err    error
		want   string
	}{
		{
			name:   "Escape",
			output: "`\\\\localhost`\n",
			want:   "```\n\\`\\\\\\\\localhost\\`\n```\n",
		},
		{
			name:   "NoEscape",
			output: "localhost\n",
			want:   "```\nlocalhost\n```\n",
		},
		{
			name:   "NoLF",
			output: "localhost",
			want:   "```\nlocalhost\n```\n",
		},
		{
			name:   "MultiLine",
			output: "localhost\nlocalhost\n",
			want:   "```\nlocalhost\nlocalhost\n```\n",
		},
		{
			name:   "EscapeErr",
			output: "`\\\\localhost`\n",
			err:    errors.New("1 + 1 = 2"),
			want:   "```\n\\`\\\\\\\\localhost\\`\n```\n1 \\+ 1 \\= 2",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := rb.Build([]byte(c.output), c.err); got != c.want {
				t.Errorf("rb.Build(%q, %v) = %q, want %q", c.output, c.err, got, c.want)
			}
		})
	}
}

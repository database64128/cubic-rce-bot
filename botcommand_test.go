package rcebot_test

import (
	"testing"

	rcebot "github.com/database64128/cubic-rce-bot"
)

func TestBotCommand(t *testing.T) {
	for _, c := range [...]struct {
		name       string
		input      string
		wantCmd    rcebot.BotCommand
		wantString string
	}{
		{
			name:  "Full",
			input: "/start@username 1 2",
			wantCmd: rcebot.BotCommand{
				Name:     "start",
				Username: "username",
				Argument: "1 2",
			},
			wantString: "/start@username 1 2",
		},
		{
			name:  "NoArgument",
			input: "/start@username",
			wantCmd: rcebot.BotCommand{
				Name:     "start",
				Username: "username",
			},
			wantString: "/start@username",
		},
		{
			name:  "NoUsername",
			input: "/start 1 2",
			wantCmd: rcebot.BotCommand{
				Name:     "start",
				Argument: "1 2",
			},
			wantString: "/start 1 2",
		},
		{
			name:  "NameOnly",
			input: "/start",
			wantCmd: rcebot.BotCommand{
				Name: "start",
			},
			wantString: "/start",
		},
		{
			name:       "Scaffolding",
			input:      "/@ ",
			wantCmd:    rcebot.BotCommand{},
			wantString: "/",
		},
		{
			name:       "NoSlash",
			input:      "!start@username 1 2",
			wantCmd:    rcebot.BotCommand{},
			wantString: "/",
		},
		{
			name:       "Empty",
			input:      "",
			wantCmd:    rcebot.BotCommand{},
			wantString: "/",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			cmd := rcebot.ParseBotCommand(c.input)
			if cmd != c.wantCmd {
				t.Errorf("ParseBotCommand(%q) = %#v, want %#v", c.input, cmd, c.wantCmd)
			}
			if gotString := cmd.String(); gotString != c.wantString {
				t.Errorf("cmd.String() = %q, want %q", gotString, c.wantString)
			}
		})
	}
}

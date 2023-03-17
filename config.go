package rcebot

import (
	"bytes"
	"context"
	"sync/atomic"
	"time"
)

const (
	// DefaultExecTimeout is the default command execution timeout.
	DefaultExecTimeout = 15 * time.Second

	// DefaultExitTimeout is the default command exit timeout.
	DefaultExitTimeout = 5 * time.Second
)

// Config is the configuration for the bot.
type Config struct {
	// Token is the bot token.
	Token string `json:"token"`

	// URL is the custom bot API URL.
	// Leave empty to use the official bot API.
	URL string `json:"url"`

	// Users is the list of authorized users.
	Users []User `json:"users"`
}

// User is an authorized user.
type User struct {
	// ID is the Telegram user ID.
	ID int64 `json:"id"`

	// Commands is the list of commands the user is allowed to execute.
	Commands []Command `json:"commands"`
}

// Command is an authorized command.
type Command struct {
	// Name is the command name.
	Name string `json:"name"`

	// Args is the list of command arguments.
	Args []string `json:"args"`

	// ExecTimeoutSec is the command execution timeout in seconds.
	// When command execution exceeds this timeout, an interrupt signal is sent to the process.
	// If the process does not exit within [ExitTimeoutSec], it is terminated.
	//
	// If zero, [DefaultExecTimeout] is used.
	ExecTimeoutSec int `json:"execTimeoutSec"`

	// ExitTimeoutSec is the command exit timeout in seconds.
	// When command execution exceeds [ExecTimeoutSec], an interrupt signal is sent to the process.
	// If the process does not exit within this timeout, it is terminated.
	//
	// If zero, [DefaultExitTimeout] is used.
	ExitTimeoutSec int `json:"exitTimeoutSec"`

	execTimeout     time.Duration
	exitTimeout     time.Duration
	cancel          atomic.Pointer[context.CancelFunc]
	outputBuffer    bytes.Buffer
	responseBuilder CommandOutputResponseBuilder
}

// UserCommandsByID returns a map of user ID to list of commands.
func (c Config) UserCommandsByID() map[int64][]Command {
	userCommandsByID := make(map[int64][]Command, len(c.Users))

	for _, user := range c.Users {
		for i := range user.Commands {
			command := &user.Commands[i]

			switch command.ExecTimeoutSec {
			case 0:
				command.execTimeout = DefaultExecTimeout
			default:
				command.execTimeout = time.Duration(command.ExecTimeoutSec) * time.Second
			}

			switch command.ExitTimeoutSec {
			case 0:
				command.exitTimeout = DefaultExitTimeout
			default:
				command.exitTimeout = time.Duration(command.ExitTimeoutSec) * time.Second
			}
		}

		userCommandsByID[user.ID] = user.Commands
	}

	return userCommandsByID
}

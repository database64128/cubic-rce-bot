package rcebot

import (
	"fmt"
	"os/exec"
	"path/filepath"
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
	Commands [][]string `json:"commands"`
}

// UserCommandsByID returns a map of user ID to list of commands.
func (c Config) UserCommandsByID() (map[int64][]exec.Cmd, error) {
	m := make(map[int64][]exec.Cmd, len(c.Users))

	for _, user := range c.Users {
		cmds := make([]exec.Cmd, len(user.Commands))

		for i, args := range user.Commands {
			if len(args) == 0 {
				return nil, fmt.Errorf("empty command for user %d", user.ID)
			}

			cmd := &cmds[i]
			cmd.Path = args[0]
			cmd.Args = args

			if filepath.Base(cmd.Path) == cmd.Path {
				lp, err := exec.LookPath(cmd.Path)
				if lp != "" {
					// Update cmd.Path even if err is non-nil.
					// If err is ErrDot (especially on Windows), lp may include a resolved
					// extension (like .exe or .bat) that should be preserved.
					cmd.Path = lp
				}
				if err != nil {
					cmd.Err = err
				}
			}
		}

		m[user.ID] = cmds
	}

	return m, nil
}

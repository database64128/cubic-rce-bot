package rcebot

import "strings"

// BotCommand represents a Telegram bot command.
//
// Field value examples are based on "/start@username 1 2".
type BotCommand struct {
	// Name is the command name.
	//
	// Example: "start"
	Name string

	// Username is the bot username, if specified.
	//
	// Example: "username"
	Username string

	// Argument is the command argument, if specified.
	//
	// Leading and trailing spaces are removed.
	//
	// Example: "1 2"
	Argument string
}

// ParseBotCommand parses a bot command from the given text.
func ParseBotCommand(text string) (cmd BotCommand) {
	if len(text) < 2 || text[0] != '/' {
		return cmd
	}

	// Remove the leading slash.
	text = text[1:]

	// Find the first space to separate the argument.
	if spaceIndex := strings.IndexByte(text, ' '); spaceIndex != -1 {
		cmd.Argument = strings.TrimSpace(text[spaceIndex+1:])
		text = text[:spaceIndex]
	}

	// Find the '@' character to separate the username.
	if atIndex := strings.IndexByte(text, '@'); atIndex != -1 {
		cmd.Name = text[:atIndex]
		cmd.Username = text[atIndex+1:]
	} else {
		cmd.Name = text
	}

	return cmd
}

// String returns the string representation of the bot command.
func (cmd BotCommand) String() string {
	var sb strings.Builder
	sb.Grow(1 + len(cmd.Name) + 1 + len(cmd.Username) + 1 + len(cmd.Argument))
	sb.WriteByte('/')
	sb.WriteString(cmd.Name)
	if cmd.Username != "" {
		sb.WriteByte('@')
		sb.WriteString(cmd.Username)
	}
	if cmd.Argument != "" {
		sb.WriteByte(' ')
		sb.WriteString(cmd.Argument)
	}
	return sb.String()
}

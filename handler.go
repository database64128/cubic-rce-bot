package rcebot

import (
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"

	tele "gopkg.in/telebot.v3"
)

var Commands = []tele.Command{
	{
		Text:        "start",
		Description: "Get started with the bot",
	},
	{
		Text:        "list",
		Description: "List commands authorized for you to request execution",
	},
	{
		Text:        "exec",
		Description: "Execute an authorized command at the specified index",
	},
}

const startTextMarkdownV2 = `This bot allows you to execute commands on the host it is running on\.
You can only execute commands authorized for your account in the configuration\.

\- To see the list of commands you can execute, use ` + "`/list`" + `\.
\- To execute a command, use ` + "`/exec <index>`" + `\.
`

// HandleStart handles the `/start` command.
func HandleStart(c tele.Context) error {
	return c.Send(startTextMarkdownV2, tele.ModeMarkdownV2)
}

// Handler handles bot commands.
type Handler struct {
	userCommandsByID atomic.Pointer[map[int64][]exec.Cmd]
	executor         Executor
	responseBuilder  CommandOutputResponseBuilder
}

// ReplaceUserCommandsByID replaces the user commands map.
func (h *Handler) ReplaceUserCommandsByID(m map[int64][]exec.Cmd) {
	h.userCommandsByID.Store(&m)
}

// HandleList handles the `/list` command.
func (h *Handler) HandleList(c tele.Context) error {
	userCommandsByID := *h.userCommandsByID.Load()
	cmds := userCommandsByID[c.Sender().ID]
	if len(cmds) == 0 {
		return c.Send("You are not authorized to execute any commands.")
	}

	var sb strings.Builder
	for i, cmd := range cmds {
		sb.WriteString("\\[")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\\] `")
		for i := 0; i < len(cmd.Args)-1; i++ {
			writeQuotedArg(&sb, cmd.Args[i])
			sb.WriteByte(' ')
		}
		writeQuotedArg(&sb, cmd.Args[len(cmd.Args)-1])
		sb.WriteString("`\n")
	}
	return c.Send(sb.String(), tele.ModeMarkdownV2)
}

func writeQuotedArg(sb *strings.Builder, arg string) {
	needQuotes := strings.IndexByte(arg, ' ') != -1
	if needQuotes {
		sb.WriteByte('\'')
	}
	sb.WriteString(EscapeMarkdownV2CodeBlock(arg))
	if needQuotes {
		sb.WriteByte('\'')
	}
}

// HandleExec handles the `/exec` command.
func (h *Handler) HandleExec(c tele.Context) error {
	userCommandsByID := *h.userCommandsByID.Load()
	cmds := userCommandsByID[c.Sender().ID]
	if len(cmds) == 0 {
		return c.Send("You are not authorized to execute any commands.")
	}

	index, err := strconv.Atoi(c.Message().Payload)
	if err != nil {
		return c.Send("Invalid index.")
	}
	if index < 0 || index >= len(cmds) {
		return c.Send("Index out of range.")
	}

	cmd := cmds[index]
	output, err := h.executor.Execute(cmd)
	resp := h.responseBuilder.Build(output, err)
	return c.Send(resp, tele.ModeMarkdownV2)
}

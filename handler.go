package rcebot

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
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
	{
		Text:        "cancel",
		Description: "Cancel a running command at the specified index",
	},
}

const startTextMarkdownV2 = `This bot allows you to execute commands on the host it is running on\.
You can only execute commands authorized for your account in the configuration\.

\- To see the list of commands you can execute, use ` + "`/list`" + `\.
\- To execute a command, use ` + "`/exec <index>`" + `\.
`

// HandleStart handles the `/start` command.
func HandleStart(c tele.Context) error {
	return c.Reply(startTextMarkdownV2, tele.ModeMarkdownV2)
}

// Handler handles bot commands.
type Handler struct {
	wg               sync.WaitGroup
	ctx              context.Context
	userCommandsByID atomic.Pointer[map[int64][]Command]
}

// Wait waits for all running commands to finish.
func (h *Handler) Wait() {
	h.wg.Wait()
}

// SetContext sets the context.
func (h *Handler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

// ReplaceUserCommandsByID replaces the user commands map.
func (h *Handler) ReplaceUserCommandsByID(m map[int64][]Command) {
	h.userCommandsByID.Store(&m)
}

// SetUserCommands is a middleware that adds the user's list of authorized commands to the context.
// It short-circuits the command handler if the user is not authorized to execute any commands.
func (h *Handler) SetUserCommands(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		userCommandsByID := *h.userCommandsByID.Load()
		commands := userCommandsByID[c.Sender().ID]
		if len(commands) == 0 {
			return c.Reply("You are not authorized to execute any commands.")
		}
		c.Set("commands", commands)
		return next(c)
	}
}

// userCommands returns the user's list of authorized commands from the context.
func userCommands(c tele.Context) []Command {
	return c.Get("commands").([]Command)
}

// HandleList handles the `/list` command.
func (h *Handler) HandleList(c tele.Context) error {
	var sb strings.Builder
	for i, command := range userCommands(c) {
		sb.WriteString("\\[")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("\\] `")
		writeQuotedArg(&sb, command.Name)
		for i := range command.Args {
			sb.WriteByte(' ')
			writeQuotedArg(&sb, command.Args[i])
		}
		sb.WriteString("`\n")
	}
	return c.Reply(sb.String(), tele.ModeMarkdownV2)
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

// SetCommand is a middleware that parses the specified command index and adds the command and its index to the context.
// It short-circuits the command handler if the index is invalid.
func (h *Handler) SetCommand(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		index, err := strconv.Atoi(c.Message().Payload)
		if err != nil || index < 0 {
			return c.Reply("Invalid index.")
		}

		commands := userCommands(c)
		if index >= len(commands) {
			return c.Reply("Index out of range\\. Use `/list` to see the list of commands\\.", tele.ModeMarkdownV2)
		}

		c.Set("index", index)
		c.Set("command", &commands[index])
		return next(c)
	}
}

// index returns the command index from the context.
func index(c tele.Context) int {
	return c.Get("index").(int)
}

// command returns the command from the context.
func command(c tele.Context) *Command {
	return c.Get("command").(*Command)
}

// HandleExec handles the `/exec` command.
func (h *Handler) HandleExec(c tele.Context) error {
	h.wg.Add(1)
	defer h.wg.Done()

	command := command(c)
	execCtx, cancel := context.WithTimeout(h.ctx, command.execTimeout)
	defer cancel()

	if !command.cancel.CompareAndSwap(nil, &cancel) {
		index := index(c)
		return c.Reply("The command is already running\\. Use `/cancel "+strconv.Itoa(index)+"` to cancel it\\.", tele.ModeMarkdownV2)
	}
	defer command.cancel.Store(nil)

	cmd := exec.CommandContext(execCtx, command.Name, command.Args...)
	cmd.Stdout = &command.outputBuffer
	cmd.Stderr = &command.outputBuffer
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = command.execTimeout

	err := cmd.Run()
	output := command.outputBuffer.Bytes()
	command.outputBuffer.Reset()

	resp := command.responseBuilder.Build(output, err)
	return c.Reply(resp, tele.ModeMarkdownV2)
}

// HandleCancel handles the `/cancel` command.
func (h *Handler) HandleCancel(c tele.Context) error {
	command := command(c)
	cancel := command.cancel.Load()
	if cancel == nil {
		index := index(c)
		return c.Reply("The command is not running\\. Use `/exec "+strconv.Itoa(index)+"` to execute it\\.", tele.ModeMarkdownV2)
	}
	(*cancel)()
	return c.Reply("The command has been canceled. You may need to wait up to " + command.exitTimeout.String() + " for it to be killed.")
}

package rcebot

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/database64128/cubic-rce-bot/tslog"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var Commands = []models.BotCommand{
	{
		Command:     "start",
		Description: "Get started with the bot",
	},
	{
		Command:     "list",
		Description: "List commands authorized for you to request execution",
	},
	{
		Command:     "exec",
		Description: "Execute an authorized command at the specified index",
	},
	{
		Command:     "cancel",
		Description: "Cancel a running command at the specified index",
	},
}

const startTextMarkdownV2 = `This bot allows you to execute commands on the host it is running on\.
You can only execute commands authorized for your account in the configuration\.

\- To see the list of commands you can execute, use ` + "`/list`" + `\.
\- To execute a command, use ` + "`/exec <index>`" + `\.
`

// handleStart handles the `/start` command.
func handleStart(ctx context.Context, b *bot.Bot, message *models.Message) error {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          message.Chat.ID,
		MessageThreadID: message.MessageThreadID,
		Text:            startTextMarkdownV2,
		ParseMode:       models.ParseModeMarkdown,
		ReplyParameters: &models.ReplyParameters{
			MessageID: message.ID,
		},
	})
	return err
}

// Handler handles bot commands.
type Handler struct {
	botUsername      string
	logger           *tslog.Logger
	wg               sync.WaitGroup
	userCommandsByID atomic.Pointer[map[int64][]Command]
	handleList       func(ctx context.Context, b *bot.Bot, message *models.Message, cmdArg string) error
	handleExec       func(ctx context.Context, b *bot.Bot, message *models.Message, cmdArg string) error
	handleCancel     func(ctx context.Context, b *bot.Bot, message *models.Message, cmdArg string) error
}

// NewHandler returns a new handler for bot commands.
func NewHandler(botUsername string, logger *tslog.Logger) *Handler {
	h := Handler{
		botUsername: botUsername,
		logger:      logger,
	}
	h.handleList = requireUserCommands(&h.userCommandsByID, handleList)
	h.handleExec = requireUserCommands(&h.userCommandsByID, requireCommandIndex(newExecHandler(&h.wg)))
	h.handleCancel = requireUserCommands(&h.userCommandsByID, requireCommandIndex(handleCancel))
	return &h
}

// Wait waits for all running commands to finish.
func (h *Handler) Wait() {
	h.wg.Wait()
}

// SetBotUsername sets the bot username.
func (h *Handler) SetBotUsername(username string) {
	h.botUsername = username
}

// ReplaceUserCommandsByID replaces the user commands map.
func (h *Handler) ReplaceUserCommandsByID(m map[int64][]Command) {
	h.userCommandsByID.Store(&m)
}

// Handle processes a bot command update.
func (h *Handler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	message := update.Message
	botCmd := ParseBotCommand(message.Text)

	// Ignore commands meant for other bots.
	if botCmd.Username != "" && botCmd.Username != h.botUsername {
		return
	}

	if h.logger.Enabled(slog.LevelDebug) {
		h.logger.Debug("Handling bot command",
			slog.Int("id", message.ID),
			slog.Int64("fromID", message.From.ID),
			slog.String("fromFirstName", message.From.FirstName),
			slog.String("fromUsername", message.From.Username),
			slog.Int64("chatID", message.Chat.ID),
			slog.String("text", message.Text),
		)
	}

	var err error
	switch botCmd.Name {
	case "start":
		err = handleStart(ctx, b, message)
	case "list":
		err = h.handleList(ctx, b, message, botCmd.Argument)
	case "exec":
		err = h.handleExec(ctx, b, message, botCmd.Argument)
	case "cancel":
		err = h.handleCancel(ctx, b, message, botCmd.Argument)
	default:
		return
	}
	if err != nil {
		h.logger.Warn("Failed to handle bot command",
			slog.Int("id", message.ID),
			slog.Int64("fromID", message.From.ID),
			slog.String("fromFirstName", message.From.FirstName),
			slog.String("fromUsername", message.From.Username),
			slog.Int64("chatID", message.Chat.ID),
			slog.String("text", message.Text),
			tslog.Err(err),
		)
		return
	}

	h.logger.Info("Handled bot command",
		slog.Int("id", message.ID),
		slog.Int64("fromID", message.From.ID),
		slog.String("fromFirstName", message.From.FirstName),
		slog.String("fromUsername", message.From.Username),
		slog.Int64("chatID", message.Chat.ID),
		slog.String("text", message.Text),
	)
}

// requireUserCommands is a middleware that adds the user's list of authorized commands to the arguments passed to
// the next handler. It short-circuits the command handler if the user is not authorized to execute any commands.
func requireUserCommands(
	userCommandsByID *atomic.Pointer[map[int64][]Command],
	next func(ctx context.Context, b *bot.Bot, message *models.Message, cmdArg string, commands []Command) error,
) func(ctx context.Context, b *bot.Bot, message *models.Message, cmdArg string) error {
	return func(ctx context.Context, b *bot.Bot, message *models.Message, cmdArg string) error {
		userCommandsByID := *userCommandsByID.Load()
		commands := userCommandsByID[message.From.ID]
		if len(commands) == 0 {
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          message.Chat.ID,
				MessageThreadID: message.MessageThreadID,
				Text:            "You are not authorized to execute any commands.",
				ReplyParameters: &models.ReplyParameters{
					MessageID: message.ID,
				},
			})
			return err
		}
		return next(ctx, b, message, cmdArg, commands)
	}
}

// handleList handles the `/list` command.
func handleList(ctx context.Context, b *bot.Bot, message *models.Message, _ string, commands []Command) error {
	var sb strings.Builder
	for i := range commands {
		command := &commands[i]
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

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          message.Chat.ID,
		MessageThreadID: message.MessageThreadID,
		Text:            sb.String(),
		ParseMode:       models.ParseModeMarkdown,
		ReplyParameters: &models.ReplyParameters{
			MessageID: message.ID,
		},
	})
	return err
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

// requireCommandIndex is a middleware that parses the bot command argument as a command index and adds it to
// the arguments passed to the next handler. It short-circuits the command handler if the index is invalid.
func requireCommandIndex(
	next func(ctx context.Context, b *bot.Bot, message *models.Message, commands []Command, index int) error,
) func(ctx context.Context, b *bot.Bot, message *models.Message, cmdArg string, commands []Command) error {
	return func(ctx context.Context, b *bot.Bot, message *models.Message, cmdArg string, commands []Command) error {
		index, err := strconv.Atoi(cmdArg)
		if err != nil || index < 0 {
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          message.Chat.ID,
				MessageThreadID: message.MessageThreadID,
				Text:            "Invalid command index.",
				ReplyParameters: &models.ReplyParameters{
					MessageID: message.ID,
				},
			})
			return err
		}

		if index >= len(commands) {
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          message.Chat.ID,
				MessageThreadID: message.MessageThreadID,
				Text:            "Index out of range\\. Use `/list` to see the list of commands\\.",
				ParseMode:       models.ParseModeMarkdown,
				ReplyParameters: &models.ReplyParameters{
					MessageID: message.ID,
				},
			})
			return err
		}

		return next(ctx, b, message, commands, index)
	}
}

// newExecHandler returns a new handler that handles the `/exec` command.
func newExecHandler(
	wg *sync.WaitGroup,
) func(ctx context.Context, b *bot.Bot, message *models.Message, commands []Command, index int) error {
	return func(ctx context.Context, b *bot.Bot, message *models.Message, commands []Command, index int) error {
		wg.Add(1)
		defer wg.Done()

		command := &commands[index]
		execCtx, cancel := context.WithTimeout(ctx, command.ExecTimeout.Value())
		defer cancel()

		if !command.cancel.CompareAndSwap(nil, &cancel) {
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:          message.Chat.ID,
				MessageThreadID: message.MessageThreadID,
				Text:            "The command is already running\\. Use `/cancel " + strconv.Itoa(index) + "` to cancel it\\.",
				ParseMode:       models.ParseModeMarkdown,
				ReplyParameters: &models.ReplyParameters{
					MessageID: message.ID,
				},
			})
			return err
		}
		defer command.cancel.Store(nil)

		cmd := exec.CommandContext(execCtx, command.Name, command.Args...)
		cmd.Stdout = &command.outputBuffer
		cmd.Stderr = &command.outputBuffer
		cmd.Cancel = func() error {
			return cmd.Process.Signal(os.Interrupt)
		}
		cmd.WaitDelay = command.ExitTimeout.Value()

		err := cmd.Run()
		output := command.outputBuffer.Bytes()
		command.outputBuffer.Reset()

		resp := command.responseBuilder.Build(output, err)
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          message.Chat.ID,
			MessageThreadID: message.MessageThreadID,
			Text:            resp,
			ParseMode:       models.ParseModeMarkdown,
			ReplyParameters: &models.ReplyParameters{
				MessageID: message.ID,
			},
		})
		return err
	}
}

// handleCancel handles the `/cancel` command.
func handleCancel(ctx context.Context, b *bot.Bot, message *models.Message, commands []Command, index int) error {
	command := &commands[index]
	cancel := command.cancel.Load()
	if cancel == nil {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:          message.Chat.ID,
			MessageThreadID: message.MessageThreadID,
			Text:            "The command is not running\\. Use `/exec " + strconv.Itoa(index) + "` to execute it\\.",
			ParseMode:       models.ParseModeMarkdown,
			ReplyParameters: &models.ReplyParameters{
				MessageID: message.ID,
			},
		})
		return err
	}
	(*cancel)()

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          message.Chat.ID,
		MessageThreadID: message.MessageThreadID,
		Text:            "The command has been canceled. You may need to wait up to " + command.ExitTimeout.Value().String() + " for it to be killed.",
		ReplyParameters: &models.ReplyParameters{
			MessageID: message.ID,
		},
	})
	return err
}

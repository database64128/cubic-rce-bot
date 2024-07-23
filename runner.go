package rcebot

import (
	"context"
	"fmt"

	"github.com/database64128/cubic-rce-bot/jsonhelper"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// Runner loads the configuration and creates a handler.
type Runner struct {
	// Config is the bot configuration.
	Config Config

	// Handler is the bot handler.
	Handler Handler

	configPath string
	logger     *zap.Logger
}

func (r *Runner) loadConfig() error {
	var config Config
	if err := jsonhelper.OpenAndDecodeDisallowUnknownFields(r.configPath, &config); err != nil {
		return err
	}

	r.Config = config
	r.Handler.ReplaceUserCommandsByID(config.UserCommandsByID())
	return nil
}

// NewRunner creates a new runner.
func NewRunner(configPath string, logger *zap.Logger) (*Runner, error) {
	r := Runner{
		configPath: configPath,
		logger:     logger,
	}
	if err := r.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &r, nil
}

func (r *Runner) onErr(err error, c tele.Context) {
	if ce := r.logger.Check(zap.WarnLevel, "Failed to handle command"); ce != nil {
		sender := c.Sender()
		ce.Write(
			zap.Int64("userID", sender.ID),
			zap.String("userFirstName", sender.FirstName),
			zap.String("username", sender.Username),
			zap.String("text", c.Text()),
			zap.Error(err),
		)
	}
}

func (r *Runner) logHandleCommand(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if ce := r.logger.Check(zap.InfoLevel, "Handling command"); ce != nil {
			sender := c.Sender()
			ce.Write(
				zap.Int64("userID", sender.ID),
				zap.String("userFirstName", sender.FirstName),
				zap.String("username", sender.Username),
				zap.String("text", c.Text()),
			)
		}
		return next(c)
	}
}

// Start starts the runner.
func (r *Runner) Start(ctx context.Context) error {
	b, err := tele.NewBot(tele.Settings{
		URL:     r.Config.URL,
		Token:   r.Config.Token,
		OnError: r.onErr,
	})
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	if err = b.SetCommands(Commands); err != nil {
		return fmt.Errorf("failed to register bot commands: %w", err)
	}

	b.Handle("/start", HandleStart, r.logHandleCommand)
	b.Handle("/list", r.Handler.HandleList, r.logHandleCommand, r.Handler.SetUserCommands)
	b.Handle("/exec", r.Handler.HandleExec, r.logHandleCommand, r.Handler.SetUserCommands, r.Handler.SetCommand)
	b.Handle("/cancel", r.Handler.HandleCancel, r.logHandleCommand, r.Handler.SetUserCommands, r.Handler.SetCommand)

	r.Handler.SetContext(ctx)
	r.registerSIGUSR1()

	go b.Start()

	go func() {
		<-ctx.Done()
		b.Stop()
	}()

	r.logger.Info("Started bot",
		zap.Int64("userID", b.Me.ID),
		zap.String("userFirstName", b.Me.FirstName),
		zap.String("username", b.Me.Username),
	)
	return nil
}

// Wait waits for the runner to finish.
func (r *Runner) Wait() {
	r.Handler.Wait()
}

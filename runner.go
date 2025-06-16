package rcebot

import (
	"context"
	"fmt"
	"time"

	"github.com/database64128/cubic-rce-bot/jsoncfg"
	"github.com/database64128/cubic-rce-bot/webhook"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// Runner loads the configuration and creates a handler.
type Runner struct {
	// Config is the bot configuration.
	Config Config

	// Handler is the bot handler.
	Handler Handler

	configPath    string
	logger        *zap.Logger
	webhookServer *webhook.Server
	bot           *tele.Bot
}

func (r *Runner) loadConfig() error {
	var config Config
	if err := jsoncfg.Open(r.configPath, &config); err != nil {
		return err
	}

	r.Config = config
	r.Handler.ReplaceUserCommandsByID(config.UserCommandsByID())
	return nil
}

// SaveConfig saves the current configuration to the file.
func (r *Runner) SaveConfig() error {
	return jsoncfg.Save(r.configPath, r.Config)
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
	var w *tele.Webhook
	s := tele.Settings{
		URL:     r.Config.URL,
		Token:   r.Config.Token,
		OnError: r.onErr,
	}

	if r.Config.Webhook.Enabled {
		w = &tele.Webhook{
			SecretToken: r.Config.Webhook.SecretToken,
			Endpoint: &tele.WebhookEndpoint{
				PublicURL: r.Config.Webhook.URL,
			},
		}
		s.Poller = w
	}

	retryOnError := func(f func() error) error {
		for {
			if err := f(); err != nil {
				r.logger.Warn("Failed to complete API request, retrying in 30 seconds", zap.Error(err))
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(30 * time.Second):
					continue
				}
			}
			break
		}
		return nil
	}

	var b *tele.Bot

	if err := retryOnError(func() (err error) {
		b, err = tele.NewBot(s)
		return err
	}); err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	r.bot = b

	if err := retryOnError(func() error {
		return b.SetCommands(Commands)
	}); err != nil {
		return fmt.Errorf("failed to set bot commands: %w", err)
	}

	if !r.Config.Webhook.Enabled {
		if err := retryOnError(func() error {
			return b.RemoveWebhook()
		}); err != nil {
			return fmt.Errorf("failed to remove webhook: %w", err)
		}
	}

	b.Handle("/start", HandleStart, r.logHandleCommand)
	b.Handle("/list", r.Handler.HandleList, r.logHandleCommand, r.Handler.SetUserCommands)
	b.Handle("/exec", r.Handler.HandleExec, r.logHandleCommand, r.Handler.SetUserCommands, r.Handler.SetCommand)
	b.Handle("/cancel", r.Handler.HandleCancel, r.logHandleCommand, r.Handler.SetUserCommands, r.Handler.SetCommand)

	r.Handler.SetContext(ctx)
	r.registerSIGUSR1()

	go b.Start()

	r.logger.Info("Started bot",
		zap.Int64("userID", b.Me.ID),
		zap.String("userFirstName", b.Me.FirstName),
		zap.String("username", b.Me.Username),
	)

	if r.Config.Webhook.Enabled {
		var err error
		r.webhookServer, err = r.Config.Webhook.NewServer(r.logger, w)
		if err != nil {
			return fmt.Errorf("failed to create webhook server: %w", err)
		}
		if err = r.webhookServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start webhook server: %w", err)
		}
	}

	return nil
}

// Stop stops the runner.
func (r *Runner) Stop() {
	// Stop the webhook server if it exists.
	if r.webhookServer != nil {
		if err := r.webhookServer.Stop(); err != nil {
			r.logger.Error("Failed to stop webhook server", zap.Error(err))
		}
	}

	// Stop the bot.
	r.bot.Stop()

	// Wait for all running commands to exit.
	r.Handler.Wait()
}

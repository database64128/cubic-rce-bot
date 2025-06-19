package rcebot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/database64128/cubic-rce-bot/jsoncfg"
	"github.com/database64128/cubic-rce-bot/tslog"
	"github.com/database64128/cubic-rce-bot/webhook"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Runner loads the configuration and creates a handler.
type Runner struct {
	configPath    string
	config        Config
	handler       *Handler
	logger        *tslog.Logger
	bot           *bot.Bot
	webhookServer *webhook.Server
}

func (r *Runner) loadConfig() error {
	var config Config
	if err := jsoncfg.Open(r.configPath, &config); err != nil {
		return err
	}

	r.config = config
	r.handler.ReplaceUserCommandsByID(config.UserCommandsByID())
	return nil
}

// SaveConfig saves the current configuration to the file.
func (r *Runner) SaveConfig() error {
	return jsoncfg.Save(r.configPath, r.config)
}

// NewRunner creates a new runner.
func NewRunner(configPath string, logger *tslog.Logger) (*Runner, error) {
	r := Runner{
		configPath: configPath,
		handler:    NewHandler("", logger),
		logger:     logger,
	}
	r.registerSIGUSR1()
	if err := r.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	opts := make([]bot.Option, 0, 6)

	if r.config.URL != "" {
		opts = append(opts, bot.WithServerURL(r.config.URL))
	}

	opts = append(opts,
		bot.WithSkipGetMe(),
		bot.WithWebhookSecretToken(r.config.Webhook.SecretToken),
		bot.WithDefaultHandler(r.handler.Handle),
		bot.WithErrorsHandler(func(err error) {
			logger.Warn("Failed to handle update", tslog.Err(err))
		}),
		bot.WithAllowedUpdates(bot.AllowedUpdates{models.AllowedUpdateMessage}),
	)

	b, err := bot.New(r.config.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	r.bot = b
	return &r, nil
}

// Start starts the runner.
func (r *Runner) Start(ctx context.Context) error {
	retryOnError := func(f func() error) error {
		for {
			if err := f(); err != nil {
				if isFatalAPIError(err) {
					return err
				}
				r.logger.Warn("Failed to complete API request, retrying in 30 seconds", tslog.Err(err))
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

	var me *models.User

	if err := retryOnError(func() (err error) {
		me, err = r.bot.GetMe(ctx)
		return err
	}); err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}

	r.handler.SetBotUsername(me.Username)

	if err := retryOnError(func() error {
		_, err := r.bot.SetMyCommands(ctx, &bot.SetMyCommandsParams{
			Commands: Commands,
		})
		return err
	}); err != nil {
		return fmt.Errorf("failed to set bot commands: %w", err)
	}

	if err := retryOnError(func() error {
		_, err := r.bot.SetWebhook(ctx, &bot.SetWebhookParams{
			URL:            r.config.Webhook.URL,
			AllowedUpdates: []string{models.AllowedUpdateMessage},
			SecretToken:    r.config.Webhook.SecretToken,
		})
		return err
	}); err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	if r.config.Webhook.Enabled {
		r.webhookServer = r.config.Webhook.NewServer(r.logger, r.bot.WebhookHandler())
		if err := r.webhookServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start webhook server: %w", err)
		}
		go r.bot.StartWebhook(ctx)
	} else {
		go r.bot.Start(ctx)
	}

	r.logger.Info("Started bot",
		slog.Int64("id", me.ID),
		slog.String("firstName", me.FirstName),
		slog.String("username", me.Username),
	)

	return nil
}

// isFatalAPIError returns whether the error is a fatal Telegram bot API error.
func isFatalAPIError(err error) bool {
	if err == nil {
		return false
	}

	for {
		e := errors.Unwrap(err)
		if e == nil {
			break
		}
		err = e
	}

	switch err {
	case bot.ErrorForbidden, bot.ErrorBadRequest, bot.ErrorUnauthorized, bot.ErrorNotFound, bot.ErrorConflict:
		return true
	default:
		return false
	}
}

// Stop stops the runner.
func (r *Runner) Stop() {
	// Stop the webhook server if it exists.
	if r.webhookServer != nil {
		if err := r.webhookServer.Stop(); err != nil {
			r.logger.Error("Failed to stop webhook server", tslog.Err(err))
		}
	}

	// Wait for all running commands to exit.
	r.handler.Wait()
}

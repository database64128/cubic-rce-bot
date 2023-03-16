package rcebot

import (
	"encoding/json"
	"strings"

	"github.com/database64128/cubic-rce-bot/mmap"
	"go.uber.org/zap"
)

// Runner loads the configuration and creates a handler.
type Runner struct {
	// Config is the bot configuration.
	Config Config

	// Handler is the bot handler.
	Handler Handler

	configPath          string
	cachedConfigContent string
	logger              *zap.Logger
}

func (r *Runner) loadConfig() error {
	content, err := mmap.ReadFile[string](r.configPath)
	if err != nil {
		return err
	}
	defer mmap.Unmap(content)

	if content == r.cachedConfigContent {
		return nil
	}

	var config Config
	sr := strings.NewReader(content)
	d := json.NewDecoder(sr)
	d.DisallowUnknownFields()
	if err = d.Decode(&config); err != nil {
		return err
	}

	userCommandsByID, err := config.UserCommandsByID()
	if err != nil {
		return err
	}

	r.Config = config
	r.cachedConfigContent = content
	r.Handler.ReplaceUserCommandsByID(userCommandsByID)
	return nil
}

// NewRunner creates a new runner.
func NewRunner(configPath string, logger *zap.Logger) (*Runner, error) {
	r := Runner{
		configPath: configPath,
		logger:     logger,
	}
	if err := r.loadConfig(); err != nil {
		return nil, err
	}
	return &r, nil
}

// Start starts the runner.
func (r *Runner) Start() {
	r.registerSIGUSR1()
}

//go:build unix

package rcebot

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func (r *Runner) registerSIGUSR1() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1)
	go func() {
		for range sigCh {
			if err := r.loadConfig(); err != nil {
				r.logger.Warn("Failed to reload config", zap.Error(err))
				continue
			}
			r.logger.Info("Reloaded config")
		}
	}()
}

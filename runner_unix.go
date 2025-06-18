//go:build unix

package rcebot

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/database64128/cubic-rce-bot/tslog"
)

func (r *Runner) registerSIGUSR1() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1)
	go func() {
		for range sigCh {
			if err := r.loadConfig(); err != nil {
				r.logger.Warn("Failed to reload config", tslog.Err(err))
				continue
			}
			r.logger.Info("Reloaded config")
		}
	}()
}

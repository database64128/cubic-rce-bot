package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	rcebot "github.com/database64128/cubic-rce-bot"
	"github.com/database64128/cubic-rce-bot/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version  bool
	fmtConf  bool
	testConf bool
	confPath string
	zapConf  string
	logLevel zapcore.Level
)

func init() {
	flag.BoolVar(&version, "version", false, "Print version information and exit")
	flag.BoolVar(&fmtConf, "fmtConf", false, "Format the configuration file")
	flag.BoolVar(&testConf, "testConf", false, "Test the configuration file and exit")
	flag.StringVar(&confPath, "confPath", "config.json", "Path to the JSON configuration file")
	flag.StringVar(&zapConf, "zapConf", "console", "Preset name or path to the JSON configuration file for building the zap logger.\nAvailable presets: console, console-nocolor, console-notime, systemd, production, development")
	flag.TextVar(&logLevel, "logLevel", zapcore.InfoLevel, "Log level for the console and systemd presets.\nAvailable levels: debug, info, warn, error, dpanic, panic, fatal")
}

func main() {
	flag.Parse()

	if version {
		if info, ok := debug.ReadBuildInfo(); ok {
			os.Stdout.WriteString(info.String())
		}
		return
	}

	logger, err := logging.NewZapLogger(zapConf, logLevel)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to build logger:", err)
		os.Exit(1)
	}
	defer logger.Sync()

	r, err := rcebot.NewRunner(confPath, logger)
	if err != nil {
		logger.Fatal("Failed to create bot runner",
			zap.String("confPath", confPath),
			zap.Error(err),
		)
	}

	if fmtConf {
		if err = r.SaveConfig(); err != nil {
			logger.Fatal("Failed to save config",
				zap.String("confPath", confPath),
				zap.Error(err),
			)
		}
		logger.Info("Formatted config file", zap.String("confPath", confPath))
	}

	if testConf {
		logger.Info("Config test OK", zap.String("confPath", confPath))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	for {
		if err = r.Start(ctx); err != nil {
			logger.Warn("Failed to start bot runner, retrying in 30 seconds", zap.Error(err))
			time.Sleep(30 * time.Second)
			continue
		}
		break
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("Received exit signal", zap.Stringer("signal", sig))
	signal.Stop(sigCh)

	cancel()
	r.Wait()
}

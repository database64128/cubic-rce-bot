package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	rcebot "github.com/database64128/cubic-rce-bot"
	"github.com/database64128/cubic-rce-bot/jsonhelper"
	"github.com/database64128/cubic-rce-bot/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	testConf bool
	confPath string
	zapConf  string
	logLevel zapcore.Level
)

func init() {
	flag.BoolVar(&testConf, "testConf", false, "Test the configuration file without starting the bot")
	flag.StringVar(&confPath, "confPath", "", "Path to JSON configuration file")
	flag.StringVar(&zapConf, "zapConf", "", "Preset name or path to JSON configuration file for building the zap logger.\nAvailable presets: console (default), systemd, production, development")
	flag.TextVar(&logLevel, "logLevel", zapcore.InvalidLevel, "Override the logger configuration's log level.\nAvailable levels: debug, info, warn, error, dpanic, panic, fatal")
}

func main() {
	flag.Parse()

	if confPath == "" {
		fmt.Fprintln(os.Stderr, "Missing -confPath <path>.")
		flag.Usage()
		os.Exit(1)
	}

	var zc zap.Config

	switch zapConf {
	case "console", "":
		zc = logging.NewProductionConsoleConfig(false)
	case "systemd":
		zc = logging.NewProductionConsoleConfig(true)
	case "production":
		zc = zap.NewProductionConfig()
	case "development":
		zc = zap.NewDevelopmentConfig()
	default:
		if err := jsonhelper.LoadAndDecodeDisallowUnknownFields(zapConf, &zc); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to load zap logger config:", err)
			os.Exit(1)
		}
	}

	if logLevel != zapcore.InvalidLevel {
		zc.Level.SetLevel(logLevel)
	}

	logger, err := zc.Build()
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

	cancel()
	r.Wait()
}

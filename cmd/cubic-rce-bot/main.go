package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	rcebot "github.com/database64128/cubic-rce-bot"
	"github.com/database64128/cubic-rce-bot/tslog"
)

var (
	version    bool
	fmtConf    bool
	testConf   bool
	logNoColor bool
	logNoTime  bool
	logKVPairs bool
	logJSON    bool
	logLevel   slog.Level
	confPath   string
)

func init() {
	flag.BoolVar(&version, "version", false, "Print version information and exit")
	flag.BoolVar(&fmtConf, "fmtConf", false, "Format the configuration file")
	flag.BoolVar(&testConf, "testConf", false, "Test the configuration file and exit")
	flag.BoolVar(&logNoColor, "logNoColor", false, "Disable colors in log output")
	flag.BoolVar(&logNoTime, "logNoTime", false, "Disable timestamps in log output")
	flag.BoolVar(&logKVPairs, "logKVPairs", false, "Use key=value pairs in log output")
	flag.BoolVar(&logJSON, "logJSON", false, "Use JSON in log output")
	flag.TextVar(&logLevel, "logLevel", slog.LevelInfo, "Log level, one of: DEBUG, INFO, WARN, ERROR")
	flag.StringVar(&confPath, "confPath", "config.json", "Path to the configuration file")
}

func main() {
	flag.Parse()

	if version {
		if info, ok := debug.ReadBuildInfo(); ok {
			os.Stdout.WriteString(info.String())
		}
		return
	}

	logCfg := tslog.Config{
		Level:          logLevel,
		NoColor:        logNoColor,
		NoTime:         logNoTime,
		UseTextHandler: logKVPairs,
		UseJSONHandler: logJSON,
	}
	logger := logCfg.NewLogger(os.Stderr)

	r, err := rcebot.NewRunner(confPath, logger)
	if err != nil {
		logger.Error("Failed to create bot runner",
			slog.String("confPath", confPath),
			tslog.Err(err),
		)
		os.Exit(1)
	}

	if fmtConf {
		if err = r.SaveConfig(); err != nil {
			logger.Error("Failed to save config",
				slog.String("confPath", confPath),
				tslog.Err(err),
			)
			os.Exit(1)
		}
		logger.Info("Formatted config file", slog.String("confPath", confPath))
	}

	if testConf {
		logger.Info("Config test OK", slog.String("confPath", confPath))
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		logger.Info("Received exit signal")
		stop()
	}()

	if err = r.Start(ctx); err != nil {
		logger.Error("Failed to start bot runner", tslog.Err(err))
		os.Exit(1)
	}

	<-ctx.Done()
	r.Stop()
}

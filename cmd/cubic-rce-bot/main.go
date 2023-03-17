package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	rcebot "github.com/database64128/cubic-rce-bot"
	"github.com/database64128/cubic-rce-bot/jsonhelper"
	"github.com/database64128/cubic-rce-bot/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	testConf = flag.Bool("testConf", false, "Test the configuration file without starting the services")
	confPath = flag.String("confPath", "", "Path to JSON configuration file")
	zapConf  = flag.String("zapConf", "", "Preset name or path to JSON configuration file for building the zap logger.\nAvailable presets: console (default), systemd, production, development")
	logLevel = flag.String("logLevel", "", "Override the logger configuration's log level.\nAvailable levels: debug, info, warn, error, dpanic, panic, fatal")
)

func main() {
	flag.Parse()

	if *confPath == "" {
		fmt.Println("Missing -confPath <path>.")
		flag.Usage()
		os.Exit(1)
	}

	var zc zap.Config

	switch *zapConf {
	case "console", "":
		zc = logging.NewProductionConsoleConfig(false)
	case "systemd":
		zc = logging.NewProductionConsoleConfig(true)
	case "production":
		zc = zap.NewProductionConfig()
	case "development":
		zc = zap.NewDevelopmentConfig()
	default:
		if err := jsonhelper.LoadAndDecodeDisallowUnknownFields(*zapConf, &zc); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if *logLevel != "" {
		l, err := zapcore.ParseLevel(*logLevel)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		zc.Level.SetLevel(l)
	}

	logger, err := zc.Build()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer logger.Sync()

	r, err := rcebot.NewRunner(*confPath, logger)
	if err != nil {
		logger.Fatal("Failed to create bot runner",
			zap.Stringp("confPath", confPath),
			zap.Error(err),
		)
	}

	if *testConf {
		logger.Info("Config test OK", zap.Stringp("confPath", confPath))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err = r.Start(ctx); err != nil {
		logger.Fatal("Failed to start bot runner", zap.Error(err))
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("Received exit signal", zap.Stringer("signal", sig))

	cancel()
	r.Wait()
}

// Command pichub 是 PicHub 图床服务的入口。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pichub/pichub/internal/bootstrap"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/logger"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfgPath := flag.String("c", "./configs/config.yaml", "config file path")
	showVer := flag.Bool("v", false, "show version")
	flag.Parse()

	if *showVer {
		fmt.Printf("pichub %s (built %s)\n", version, buildTime)
		return 0
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}

	log := logger.New(&cfg.Log)
	defer log.Sync()

	log.Info("starting pichub",
		logger.String("version", version),
		logger.String("built", buildTime),
		logger.String("config", cfg.LoadedPath()))
	for _, w := range cfg.Warnings() {
		log.Warn("config warning", logger.String("detail", w))
	}

	app, err := bootstrap.New(cfg, log)
	if err != nil {
		log.Error("bootstrap failed", logger.Err(err))
		return 1
	}

	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	exitCode := 0
	select {
	case sig := <-quit:
		log.Info("shutting down", logger.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			log.Error("server exited", logger.Err(err))
			exitCode = 1
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		log.Warn("shutdown incomplete", logger.Err(err))
	}

	select {
	case err := <-errCh:
		if err != nil && exitCode == 0 {
			log.Error("server exited", logger.Err(err))
			exitCode = 1
		}
	case <-time.After(5 * time.Second):
		log.Warn("wait for server exit timed out")
	}
	log.Info("bye")
	return exitCode
}

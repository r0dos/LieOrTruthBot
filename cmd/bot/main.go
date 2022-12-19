package main

import (
	"LieOrTruthBot/internal/config"
	"LieOrTruthBot/internal/provider/storage"
	"LieOrTruthBot/internal/service/bot"
	"LieOrTruthBot/pkg/log"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/yaml.v3"
)

const (
	configPath = "config.yml"

	pollerTimeout = 10 * time.Second

	envDBPath = "DB_URL"
)

func main() {
	// Cancel context if got Ctrl+C signal.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		// Run Cleanup
		cancel()
		log.Debug("Catch cancel...")
	}()

	if err := run(ctx); err != nil {
		log.Fatal("run", zap.Error(err))
	}
}

func run(ctx context.Context) error {
	log.Initialize()
	defer log.Sync()

	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get project work dir: %v", err)
	}

	path := filepath.Join(pwd, configPath)

	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file path %s: %v", path, err)
	}

	cfg := &config.Config{}
	if err := yaml.Unmarshal(bytes, cfg); err != nil {
		return fmt.Errorf("unmarshal config: %v", err)
	}

	db, err := initDB(os.Getenv(envDBPath))
	if err != nil {
		return fmt.Errorf("init db: %v", err)
	}

	defer func() {
		_ = db.Close()
	}()

	stor, err := storage.NewStorage(db)
	if err != nil {
		return fmt.Errorf("init storage: %v", err)
	}

	pref := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: pollerTimeout},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return fmt.Errorf("init telebot: %v", err)
	}

	service := bot.NewMeBot(cfg, b, stor)

	go service.Start()
	defer service.Close()

	<-ctx.Done()

	return nil
}

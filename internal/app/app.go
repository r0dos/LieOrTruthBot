package app

import (
	"LieOrTruthBot/internal/config"
	"LieOrTruthBot/internal/repository/storage"
	"LieOrTruthBot/internal/service/bot"
	"LieOrTruthBot/pkg/log"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const (
	configPath = "configs/config.yml"

	pollerTimeout = 10 * time.Second

	envDBPath = "DB_URL"
)

func App() {
	// Cancel context if got Ctrl+C signal.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		// Run Cleanup
		cancel()
		log.Debug("Catch cancel...")
	}()

	if err := run(ctx); err != nil {
		log.Panic("run", zap.Error(err))
	}
}

func run(ctx context.Context) error {
	log.Initialize()
	defer log.Sync()

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
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

	service := bot.NewLoTBot(cfg, b, stor)

	go service.Start()
	defer service.Close()

	<-ctx.Done()

	return nil
}

package bot

import (
	"LieOrTruthBot/internal/config"
	"LieOrTruthBot/internal/models/dto"
	"LieOrTruthBot/pkg/log"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/telebot.v3"
)

type Storage interface {
	AddAdmin(userID int64) error
	CheckAdmin(userID int64) (bool, error)
	AddQuestion(question string, answer bool, userID string) error
	GetTop(chatID, limit int64) ([]dto.ChartItem, error)
	GetQuestion() (string, bool, error)
	IncValue(chatID, userID int64) error
}

type MeBot struct {
	cfg     *config.Config
	bot     *telebot.Bot
	storage Storage
	rounds  map[int64]*entry
	mu      sync.RWMutex
}

func NewMeBot(cfg *config.Config, b *telebot.Bot, s Storage) *MeBot {
	me := &MeBot{
		cfg:     cfg,
		bot:     b,
		storage: s,
		rounds:  make(map[int64]*entry),
	}

	me.registerMiddlewares()
	me.registerHandlers()

	return me
}

func (m *MeBot) Start() {
	m.bot.Start()
}

func (m *MeBot) Close() {
	for i := 0; i < 5; i++ {
		c, err := m.bot.Close()
		if err != nil {
			log.Error("bot close", zap.Error(err))
		}

		if c {
			return
		}

		time.Sleep(time.Second)
	}
}

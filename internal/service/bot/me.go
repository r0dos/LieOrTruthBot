package bot

import (
	"LieOrTruthBot/internal/config"
	"LieOrTruthBot/internal/models/dto"
	"LieOrTruthBot/pkg/log"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/telebot.v3"
)

type Storage interface {
	AddAdmin(userID int64) error
	CheckAdmin(userID int64) (bool, error)
	AddQuestion(question string, answer bool, detailed, userID string) error
	GetTop(chatID, limit int64) ([]dto.ChartItem, error)
	GetQuestion() (string, bool, string, error)
	IncValue(chatID, userID int64) error
}

type LoTBot struct {
	cfg      *config.Config
	bot      *telebot.Bot
	storage  Storage
	rounds   map[int64]*round
	waitText map[int64]*Entry
	mu       sync.RWMutex
}

type TextAction string

const (
	waitQuestion TextAction = "q"
	waitDetailed TextAction = "d"
)

type Entry struct {
	Action   TextAction
	Question string
	Answer   bool
	Detailed string
}

func (e *Entry) String() string {
	answer := "Ложь"
	if e.Answer {
		answer = "Правда"
	}

	text := fmt.Sprintf(" Вопрос:\n%s\n\nПравильный ответ: %s", e.Question, answer)

	if e.Detailed != "" {
		text += fmt.Sprintf("\n\nРазвернутый ответ:\n%s", e.Detailed)
	}

	return text
}

func NewLoTBot(cfg *config.Config, b *telebot.Bot, s Storage) *LoTBot {
	me := &LoTBot{
		cfg:      cfg,
		bot:      b,
		storage:  s,
		rounds:   make(map[int64]*round),
		waitText: make(map[int64]*Entry),
	}

	me.registerMiddlewares()
	me.registerHandlers()

	return me
}

func (l *LoTBot) Start() {
	l.bot.Start()
}

func (l *LoTBot) Close() {
	for i := 0; i < 5; i++ {
		c, err := l.bot.Close()
		if err != nil {
			log.Error("bot close", zap.Error(err))
		}

		if c {
			return
		}

		time.Sleep(time.Second)
	}
}

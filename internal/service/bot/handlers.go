package bot

import (
	"LieOrTruthBot/pkg/log"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/telebot.v3"
)

const topLimit = 10

const roundTime = 2 * time.Minute

type voteFunc func(userID int64, v bool)

type entry struct {
	cancel context.CancelFunc
	vote   voteFunc
}

func (m *MeBot) registerHandlers() {
	// Command: /ping
	m.bot.Handle("/ping", m.handlerPong)

	// Command: /id
	m.bot.Handle("/id", m.handlerID)

	// Command: /admin
	m.bot.Handle("/admin", m.handlerAddAdmin)

	// Command: /question
	m.bot.Handle("/question", m.handlerQuestion)
	m.bot.Handle("\fanswer", m.handlerAnswer)

	groupOnly := m.bot.Group()
	groupOnly.Use(middlewareFromGroup)

	//Command: /top
	groupOnly.Handle("/top", m.handlerTop)

	//Command: /round
	groupOnly.Handle("/round", m.handlerRound)
	m.bot.Handle("\fvote", m.handlerVote)

	//Command: /help
	groupOnly.Handle("/help", m.handlerHelp)

}

func (m *MeBot) handlerHelp(c telebot.Context) error {
	_, err := c.Bot().Reply(c.Message(),
		"Список команд:\n"+
			"\t/round - запуск нового раунда\n"+
			"\t/top - топ 10 играков\n"+
			"\nДля добавления вопроса:\n"+
			"\t- получить свой ID отправив боту в личку команду /id\n"+
			"\t- попросите владельца добавить ваш id для достука к добавлению вопроса\n"+
			"\t- добавте вопрос отправив боту в личку \"/question <ваш вопрос>\"",
		telebot.ModeMarkdown)

	return err
}

func (m *MeBot) handlerPong(c telebot.Context) error {
	return c.Send("pong")
}

func (m *MeBot) handlerID(c telebot.Context) error {
	return c.Send(fmt.Sprintf("Your Telegram Chat ID is: %d", c.Chat().ID))
}

func (m *MeBot) handlerAddAdmin(c telebot.Context) error {
	if c.Message().Sender.ID != m.cfg.SuperUser {
		return nil
	}

	for _, item := range c.Args() {
		n, err := strconv.ParseInt(item, 10, 64)
		if err != nil {
			log.Error("add admin pars int64", zap.Error(err))

			continue
		}

		if err := m.storage.AddAdmin(n); err != nil {
			log.Error("add admin storage", zap.Error(err))
		}
	}

	return nil
}

func (m *MeBot) handlerQuestion(c telebot.Context) error {
	isAdmin, err := m.storage.CheckAdmin(c.Message().Sender.ID)
	if err != nil {
		return fmt.Errorf("check admin %v", err)
	}

	if !isAdmin && m.cfg.SuperUser != c.Message().Sender.ID {
		return nil
	}

	if len(c.Args()) == 0 {
		return c.Send("отправте вопрос в формате \"/question <ваш вопрос>\"")
	}

	question := strings.Join(c.Args(), " ")

	keyboard := &telebot.ReplyMarkup{}
	row := telebot.Row{
		keyboard.Data("Правда", "answer", "1", question, c.Message().Sender.Recipient()),
		keyboard.Data("Ложь", "answer", "0", question, c.Message().Sender.Recipient()),
		keyboard.Data("Отмена", "answer", "cancel"),
	}
	keyboard.Inline(row)

	return c.Send(fmt.Sprintf("Какой правильный ответ на вопрос:\n\n%s", question), keyboard)
}

func (m *MeBot) handlerAnswer(c telebot.Context) error {
	if err := c.Bot().Delete(c.Message()); err != nil {
		log.Error("delete message question", zap.Error(err))
	}

	if c.Data() == "cancel" {
		return nil
	}

	data := strings.Split(c.Data(), "|")

	if len(data) < 3 {
		return c.Send("что-то пошло не так...")
	}

	boolValue, err := strconv.ParseBool(data[0])
	if err != nil {
		log.Error("pars bool", zap.Error(err))

		return c.Send("что-то пошло не так...")
	}

	if err := m.storage.AddQuestion(data[1], boolValue, data[2]); err != nil {
		log.Error("add question", zap.Error(err))

		return c.Send("что-то пошло не так...")
	}

	return c.Send("вопрос добавлен")
}

func (m *MeBot) handlerTop(c telebot.Context) error {
	top, err := m.storage.GetTop(c.Message().Chat.ID, topLimit)
	if err != nil {
		return fmt.Errorf("get top: %v", err)
	}

	var text strings.Builder

	_, err = fmt.Fprintf(&text, "Top %d игроков:", topLimit)
	if err != nil {
		return fmt.Errorf("string build: %v", err)
	}

	for _, item := range top {
		member, err := c.Bot().ChatMemberOf(c.Chat(), &telebot.User{ID: item.UserID})
		if err != nil {
			log.Error("get member info", zap.Error(err))

			continue
		}

		_, err = fmt.Fprintf(&text, "\n%s - %d", getUsername(member.User), item.Value)
		if err != nil {
			return fmt.Errorf("string build: %v", err)
		}
	}

	return c.Send(text.String())
}

func (m *MeBot) handlerRound(c telebot.Context) error {
	m.mu.RLock()
	if _, ok := m.rounds[c.Chat().ID]; ok {
		m.mu.RUnlock()

		return c.Send("Раунд уже идёт")
	}
	m.mu.RUnlock()

	question, answer, err := m.storage.GetQuestion()
	if err != nil {
		return fmt.Errorf("get question: %v", err)
	}

	keyboard := &telebot.ReplyMarkup{}
	row := telebot.Row{
		keyboard.Data("Правда", "vote", "1"),
		keyboard.Data("Ложь", "vote", "0"),
	}
	keyboard.Inline(row)

	msg, err := m.bot.Send(c.Chat(), question, keyboard)
	if err != nil {
		return fmt.Errorf("send question: %v", err)
	}

	var mu sync.RWMutex

	answers := make(map[int64]bool)
	vote := func(userID int64, v bool) {
		mu.Lock()
		defer mu.Unlock()

		answers[userID] = v
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		timer := time.NewTimer(roundTime)
		defer timer.Stop()

		select {
		case <-ctx.Done():
		case <-timer.C:
		}

		if err := m.bot.Delete(msg); err != nil {
			log.Error("delete question message", zap.Error(err))
		}

		if len(answers) == 0 {
			if _, err := m.bot.Send(c.Chat(), "В этом раунде нет активных участников"); err != nil {
				log.Error("send result round", zap.Error(err))
			}

			return
		}

		var (
			right strings.Builder
			wrong strings.Builder
		)

		mu.RLock()
		for uID, ans := range answers {
			member, err := c.Bot().ChatMemberOf(c.Chat(), &telebot.User{ID: uID})
			if err != nil {
				log.Error("get member info", zap.Error(err))

				continue
			}

			if ans == answer {
				if err := m.storage.IncValue(c.Chat().ID, uID); err != nil {
					log.Error("inc pointer", zap.Error(err))
				}

				_, _ = fmt.Fprintf(&right, "\n%s", getUsername(member.User))

				continue
			}

			_, _ = fmt.Fprintf(&wrong, "\n%s", getUsername(member.User))
		}
		mu.RUnlock()

		if _, err := m.bot.Send(c.Chat(),
			fmt.Sprintf("Правильно ответили:%s\n\nНеправильно ответили:%s", right.String(), wrong.String()),
		); err != nil {
			log.Error("send result round", zap.Error(err))
		}

		m.mu.Lock()
		delete(m.rounds, c.Chat().ID)
		m.mu.Unlock()
	}()

	m.mu.Lock()
	m.rounds[c.Chat().ID] = &entry{
		cancel: cancelFunc,
		vote:   vote,
	}
	m.mu.Unlock()

	return nil
}

func (m *MeBot) handlerVote(c telebot.Context) error {
	boolValue, err := strconv.ParseBool(c.Data())
	if err != nil {
		return fmt.Errorf("pars bool: %v", err)
	}

	m.mu.RLock()
	if e, ok := m.rounds[c.Chat().ID]; ok {
		e.vote(c.Sender().ID, boolValue)
	}
	m.mu.RUnlock()

	return nil
}

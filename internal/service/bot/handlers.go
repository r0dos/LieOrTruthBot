package bot

import (
	"LieOrTruthBot/pkg/log"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/telebot.v3"
)

const topLimit = 10

const roundTime = 30 * time.Second

type voteFunc func(userID int64, v bool)

type round struct {
	cancel context.CancelFunc
	vote   voteFunc
}

func (l *LoTBot) registerHandlers() {
	// Command: /ping
	l.bot.Handle("/ping", l.handlerPong)

	pmOnly := l.bot.Group()
	pmOnly.Use(middlewareFromPM)

	// Command: /id
	pmOnly.Handle("/id", l.handlerID)

	// Command: /admin
	pmOnly.Handle("/admin", l.handlerAddAdmin)

	// Command: /question
	pmOnly.Handle("/question", l.handlerQuestion)
	pmOnly.Handle(telebot.OnText, l.handlerOnText)
	pmOnly.Handle("\fcancel", l.handlerCancel)
	pmOnly.Handle("\fanswer", l.handlerAnswer)
	pmOnly.Handle("\fadd", l.handlerAdd)
	pmOnly.Handle("\fdetailed", l.handlerDetailed)

	groupOnly := l.bot.Group()
	groupOnly.Use(middlewareFromGroup)

	//Command: /top
	groupOnly.Handle("/top", l.handlerTop)

	//Command: /round
	groupOnly.Handle("/round", l.handlerRound)
	l.bot.Handle("\fvote", l.handlerVote)

	//Command: /help
	l.bot.Handle("/help", l.handlerHelp)
}

func (l *LoTBot) handlerHelp(c telebot.Context) error {
	_, err := c.Bot().Reply(c.Message(),
		"Список команд:\n"+
			" /round - запуск нового раунда\n"+
			" /top - топ 10 игроков\n"+
			"\nДля добавления вопроса:\n"+
			" - попросите владельца добавить ваш id для достука к добавлению вопроса\n"+
			" - добавте вопрос отправив боту в личку /question",
		telebot.ModeMarkdown)

	return err
}

func (l *LoTBot) handlerPong(c telebot.Context) error {
	return c.Send("pong")
}

func (l *LoTBot) handlerID(c telebot.Context) error {
	return c.Send(fmt.Sprintf("Your Telegram Chat ID is: %d", c.Chat().ID))
}

func (l *LoTBot) handlerAddAdmin(c telebot.Context) error {
	if c.Message().Sender.ID != l.cfg.SuperUser {
		return nil
	}

	if c.Message().FromGroup() && c.Message().ReplyTo != nil {
		if err := l.storage.AddAdmin(c.Message().ReplyTo.Sender.ID); err != nil {
			return fmt.Errorf("add admin storage: %v", err)
		}

		return c.Send(fmt.Sprintf("Добавил %s в список админов", getUsername(c.Message().ReplyTo.Sender)))
	}

	for _, item := range c.Args() {
		n, err := strconv.ParseInt(item, 10, 64)
		if err != nil {
			log.Error("add admin pars int64", zap.Error(err))

			continue
		}

		if err := l.storage.AddAdmin(n); err != nil {
			log.Error("add admin storage", zap.Error(err))
		}
	}

	return nil
}

func (l *LoTBot) handlerQuestion(c telebot.Context) error {
	isAdmin, err := l.storage.CheckAdmin(c.Message().Sender.ID)
	if err != nil {
		return fmt.Errorf("check admin %v", err)
	}

	if !isAdmin && l.cfg.SuperUser != c.Message().Sender.ID {
		return nil
	}

	l.mu.Lock()
	l.waitText[c.Message().Sender.ID] = &Entry{
		Action: waitQuestion,
	}
	l.mu.Unlock()

	return c.Send("Ожидаю вопрос в следующем сообщении")
}

func (l *LoTBot) handlerOnText(c telebot.Context) error {
	l.mu.Lock()
	act, ok := l.waitText[c.Message().Sender.ID]
	l.mu.Unlock()

	if !ok {
		return nil
	}

	switch act.Action {
	case waitQuestion:
		act.Question = c.Message().Text

		keyboard := &telebot.ReplyMarkup{}
		row := telebot.Row{
			keyboard.Data("Правда", "answer", "1"),
			keyboard.Data("Ложь", "answer", "0"),
			keyboard.Data("Отмена", "cancel"),
		}
		keyboard.Inline(row)

		return c.Send(fmt.Sprintf("Какой правильный ответ на вопрос:\n\n%s", c.Message().Text), keyboard)
	case waitDetailed:
		act.Detailed = c.Message().Text

		keyboard := &telebot.ReplyMarkup{}
		row := telebot.Row{
			keyboard.Data("Добавить", "add"),
			keyboard.Data("Отмена", "cancel"),
		}
		keyboard.Inline(row)

		return c.Send(act.String(), keyboard)

	default:
		log.Error("untyped action")
	}

	return nil
}

func (l *LoTBot) handlerAnswer(c telebot.Context) error {
	boolValue, err := strconv.ParseBool(c.Data())
	if err != nil {
		log.Error("pars bool", zap.Error(err))

		return c.Send("что-то пошло не так...")
	}

	l.mu.Lock()
	act, ok := l.waitText[c.Sender().ID]
	l.mu.Unlock()

	if !ok {
		return errors.New("answer: not found wait")
	}

	act.Answer = boolValue

	keyboard := &telebot.ReplyMarkup{}
	row := telebot.Row{
		keyboard.Data("Развернутый ответ", "detailed"),
		keyboard.Data("Добавить", "add"),
		keyboard.Data("Отмена", "cancel"),
	}
	keyboard.Inline(row)

	return c.Edit(act.String(), keyboard)
}

func (l *LoTBot) handlerCancel(c telebot.Context) error {
	if err := c.Bot().Delete(c.Message()); err != nil {
		log.Error("delete message question", zap.Error(err))
	}

	l.mu.Lock()
	delete(l.waitText, c.Sender().ID)
	l.mu.Unlock()

	return nil
}

func (l *LoTBot) handlerAdd(c telebot.Context) error {
	l.mu.Lock()
	act, ok := l.waitText[c.Sender().ID]
	if !ok {
		l.mu.Unlock()

		return errors.New("answer: not found wait")
	}

	delete(l.waitText, c.Sender().ID)
	l.mu.Unlock()

	if err := l.storage.AddQuestion(act.Question, act.Answer, act.Detailed, fmt.Sprint(c.Sender().ID)); err != nil {
		log.Error("add question", zap.Error(err))

		return c.Send("что-то пошло не так...")
	}

	return c.Edit("Вопрос добавлен")
}

func (l *LoTBot) handlerDetailed(c telebot.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	act, ok := l.waitText[c.Sender().ID]
	if !ok {
		return errors.New("answer: not found wait")
	}

	act.Action = waitDetailed

	return c.Edit("Ожидаю развернутый ответ в следующем сообщении")
}

func (l *LoTBot) handlerTop(c telebot.Context) error {
	top, err := l.storage.GetTop(c.Message().Chat.ID, topLimit)
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

		_, err = fmt.Fprintf(&text, "\n%s - %d", getName(member.User), item.Value)
		if err != nil {
			return fmt.Errorf("string build: %v", err)
		}
	}

	return c.Send(text.String())
}

//nolint:funlen
func (l *LoTBot) handlerRound(c telebot.Context) error {
	l.mu.RLock()
	if _, ok := l.rounds[c.Chat().ID]; ok {
		l.mu.RUnlock()

		return c.Send("Раунд уже идёт")
	}
	l.mu.RUnlock()

	question, answer, detailed, err := l.storage.GetQuestion()
	if err != nil {
		return fmt.Errorf("get question: %v", err)
	}

	keyboard := &telebot.ReplyMarkup{}
	row := telebot.Row{
		keyboard.Data("Правда", "vote", "1"),
		keyboard.Data("Ложь", "vote", "0"),
	}
	keyboard.Inline(row)

	msg, err := l.bot.Send(c.Chat(), question, keyboard)
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

		if err := l.bot.Delete(msg); err != nil {
			log.Error("delete question message", zap.Error(err))
		}

		if len(answers) == 0 {
			if _, err := l.bot.Send(c.Chat(), "В этом раунде нет активных участников"); err != nil {
				log.Error("send result round", zap.Error(err))
			}

			return
		}

		var (
			right        strings.Builder
			wrong        strings.Builder
			textDetailed string
		)

		mu.RLock()
		for uID, ans := range answers {
			member, err := c.Bot().ChatMemberOf(c.Chat(), &telebot.User{ID: uID})
			if err != nil {
				log.Error("get member info", zap.Error(err))

				continue
			}

			if ans == answer {
				if err := l.storage.IncValue(c.Chat().ID, uID); err != nil {
					log.Error("inc pointer", zap.Error(err))
				}

				_, _ = fmt.Fprintf(&right, "\n%s", getUsername(member.User))

				continue
			}

			_, _ = fmt.Fprintf(&wrong, "\n%s", getUsername(member.User))
		}
		mu.RUnlock()

		correctAnswer := "Ложь"
		if answer {
			correctAnswer = "Правда"
		}

		if detailed != "" {
			textDetailed = fmt.Sprintf("\n\n%s", detailed)
		}

		if _, err := l.bot.Send(c.Chat(),
			fmt.Sprintf("||%s||\n\nПравильный ответ: ||%s||%s\n\nПравильно ответили:%s\n\nНеправильно ответили:%s",
				escapedCharacter(question), correctAnswer, textDetailed, right.String(), wrong.String()),
			telebot.ModeMarkdownV2,
		); err != nil {
			log.Error("send result round", zap.Error(err))
		}

		l.mu.Lock()
		delete(l.rounds, c.Chat().ID)
		l.mu.Unlock()
	}()

	l.mu.Lock()
	l.rounds[c.Chat().ID] = &round{
		cancel: cancelFunc,
		vote:   vote,
	}
	l.mu.Unlock()

	return nil
}

func (l *LoTBot) handlerVote(c telebot.Context) error {
	boolValue, err := strconv.ParseBool(c.Data())
	if err != nil {
		return fmt.Errorf("pars bool: %v", err)
	}

	l.mu.RLock()
	if e, ok := l.rounds[c.Chat().ID]; ok {
		e.vote(c.Sender().ID, boolValue)
	}
	l.mu.RUnlock()

	return nil
}

package bot

import (
	"LieOrTruthBot/pkg/log"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

func (l *LoTBot) registerMiddlewares() {
	l.bot.Use(middleware.Recover(func(err error) {
		log.Error("me bot in panic", zap.Error(err))
	}), middleware.AutoRespond())
}

func middlewareFromGroup(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if !c.Message().FromGroup() {
			return nil
		}

		return next(c)
	}
}

func middlewareFromPM(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Message().FromGroup() || c.Message().FromChannel() {
			return nil
		}

		return next(c)
	}
}

func middlewareCheckAdmins(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		admins, err := c.Bot().AdminsOf(c.Chat())
		if err != nil {
			return fmt.Errorf("get admins for chatID %d: %v", c.Chat().ID, err)
		}

		if !isAdmin(c.Sender(), admins) {
			msg, err := c.Bot().Reply(c.Message(), "Ты ещё слишком мал для таких дел", telebot.ModeMarkdown)
			if err != nil {
				return fmt.Errorf("create reply message: %v", err)
			}

			return c.Send(msg)
		}

		return next(c)
	}
}

func middlewareCheckAdminsReplyTo(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		admins, err := c.Bot().AdminsOf(c.Chat())
		if err != nil {
			return fmt.Errorf("get admins for chatID %d: %v", c.Chat().ID, err)
		}

		if c.Message().ReplyTo != nil && isAdmin(c.Message().ReplyTo.Sender, admins) {
			msg, err := c.Bot().Reply(c.Message().ReplyTo, "Извинись...", telebot.ModeMarkdown)
			if err != nil {
				return fmt.Errorf("create reply message: %v", err)
			}

			return c.Send(msg)
		}

		return next(c)
	}
}

func middlewareNotAdmin(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		admins, err := c.Bot().AdminsOf(c.Chat())
		if err != nil {
			return fmt.Errorf("get admins for chatID %d: %v", c.Chat().ID, err)
		}

		if isAdmin(c.Sender(), admins) {
			return nil
		}

		return next(c)
	}

}

func isAdmin(user *telebot.User, admins []telebot.ChatMember) bool {
	for _, admin := range admins {
		if user.ID == admin.User.ID {
			return true
		}
	}

	return false
}

package bot

import (
	"fmt"

	"gopkg.in/telebot.v3"
)

func getUsername(user *telebot.User) (userName string) {
	if user == nil {
		return
	}

	if user.Username != "" {
		return fmt.Sprintf("@%s", user.Username)
	}

	return getName(user)
}

func getName(user *telebot.User) (userName string) {
	if user == nil {
		return
	}

	if user.FirstName != "" {
		userName += user.FirstName
	}

	if user.LastName != "" {
		userName += " " + user.LastName
	}

	return
}

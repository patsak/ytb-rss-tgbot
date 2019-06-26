package ytbrss

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
)

type UserError struct {
	msg string
}

func WrapUserError(err error) *UserError {
	return &UserError{
		msg: err.Error(),
	}
}

func (e *UserError) Error() string {
	return  e.msg
}

func Error(b *tgbotapi.BotAPI, chatID int64,  e error) {
	userError, ok := e.(*UserError)
	if !ok {
		logrus.Error(e)
		msgConfig := tgbotapi.NewMessage(chatID, "Internal error. So sad :(")
		if _, err := b.Send(msgConfig); err != nil {
			logrus.Error(err)
		}
	}
	msgConfig := tgbotapi.NewMessage(chatID, userError.Error())
	if _, err := b.Send(msgConfig); err != nil {
		logrus.Error(err)
	}
}

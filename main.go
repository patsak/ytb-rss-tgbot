package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	ytbrss "github.com/patsak/ytb-rss-tgbot/src"
)

var (
	destDir   = flag.String("dest", "content", "destination dir")
	rssDir    = flag.String("rssDir", "rss", "rss dir")
	token     = flag.String("token", "", "bot token")
	urlPrefix = flag.String("url", "", "url prefix")
)

func main() {
	flag.Parse()

	if len(*token) == 0 {
		*token = os.Getenv("TOKEN")
	}

	bot, err := tgbotapi.NewBotAPI(*token)
	if err != nil {
		panic(err)
	}

	bot.Debug = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	fmt.Println("bot started")

	videoProcessingDialog, err := ytbrss.NewVideoProcessingDialog(*destDir)
	if err != nil {
		panic(err)
	}

	mainContext := context.Background()
	for update := range updates {
		if update.Message == nil {
			continue
		}

		cmd := update.Message.Command()
		var err error
		switch cmd {
		case "", "youtube":
			err = videoProcessingDialog.Handle(mainContext, bot, update.Message)
		}

		if err != nil {
			ytbrss.Error(bot, update.Message.Chat.ID, err)
		}
	}
}



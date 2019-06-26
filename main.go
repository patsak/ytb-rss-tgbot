package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	ytbrss "github.com/patsak/ytb-rss-tgbot/src"
	"github.com/sirupsen/logrus"
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

	encoder, err := ytbrss.NewEncoder(*destDir)
	if err != nil {
		panic(err)
	}
	mainContext := context.Background()
	for update := range updates {
		if update.Message == nil {
			continue
		}
		ctx, cancelFunc := context.WithCancel(mainContext)

		url := &url.URL{}
		url, err = url.Parse(update.Message.Text)
		if err != nil {
			ytbrss.Error(bot, update.Message.Chat.ID, err)
			continue
		}

		encodeRes, err := encoder.GetYoutubeProcessor(url)
		if err != nil {
			ytbrss.Error(bot, update.Message.Chat.ID, err)
			continue
		}

		go func() {
			ticker := time.Tick(100 * time.Millisecond)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "File size: 0")
			retMsg, err := bot.Send(msg)
			if err != nil {
				ytbrss.Error(bot, update.Message.Chat.ID, err)
				return
			}

			for {
				select {
				case <-ticker:
					progress := encodeRes.Progress()
					edit := tgbotapi.NewEditMessageText(update.Message.Chat.ID, retMsg.MessageID, fmt.Sprintf("File size: %d", progress))
					if _, err := bot.Send(edit); err != nil {
						logrus.Error(err)
					}

				case <-ctx.Done():
					edit := tgbotapi.NewEditMessageText(update.Message.Chat.ID, retMsg.MessageID, "Processing finished. Wait audio")
					if _, err := bot.Send(edit); err != nil {
						logrus.Error(err)
					}
					return
				}
			}
		}()
		err = encodeRes.Run()
		cancelFunc()
		if err != nil {
			ytbrss.Error(bot, update.Message.Chat.ID, err)
			continue
		}

		sendFile, err := os.OpenFile(encodeRes.AudioPath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			ytbrss.Error(bot, update.Message.Chat.ID, err)
			continue
		}
		reader := tgbotapi.FileReader{
			encodeRes.Title,
			sendFile,
			-1,
		}
		config := tgbotapi.NewAudioUpload(update.Message.Chat.ID, reader)

		_, err = bot.Send(config)
		if err != nil {
			ytbrss.Error(bot, update.Message.Chat.ID, err)
		}
	}

}

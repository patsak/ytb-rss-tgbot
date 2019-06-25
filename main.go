package  main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rylio/ytdl"
	"github.com/sirupsen/logrus"
)

var (
	destDir = flag.String("dest", "content", "destination dir")
	token = flag.String("token", "", "bot token")
)

func main() {
	flag.Parse()

	bot, err := tgbotapi.NewBotAPI(*token)
	if err != nil {
		panic(err)
	}

	bot.Debug = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	fmt.Println("bot started")
	for update := range updates {
		if update.Message == nil {
			continue
		}
		url := &url.URL{}
		url, err = url.Parse(update.Message.Text)
		if err != nil {
			logrus.Error(err.Error())
		}
		info, err := ytdl.GetVideoInfoFromURL(url)
		if err != nil {
			logrus.Error(err.Error())
			continue
		}
		file, err := os.Create(destVideo(info.ID))
		if err != nil {
			logrus.Error(err.Error())
			continue
		}
		err = info.Download(info.Formats[0], file)
		_ = file.Close()
		if err != nil {
			logrus.Error(err.Error())
			continue
		}

		cmd := exec.Command("ffmpeg", "-i", destVideo(info.ID), "-f", "mp3", "-y", "-ab", "64000", "-vn", destAudio(info.ID))
		output, err := cmd.CombinedOutput()
		if err != nil {
			logrus.Println("%s", string(output))
			logrus.Error(err.Error())
			continue
		}
		sendFile, err := os.OpenFile(destAudio(info.ID), os.O_RDONLY, os.ModePerm)
		if err != nil {
			logrus.Error(err.Error())
			continue
		}
		reader := tgbotapi.FileReader{
			info.Title,
			sendFile,
			-1,

		}
		config := tgbotapi.NewAudioUpload(update.Message.Chat.ID, reader)

		_, err = bot.Send(config)
		if err != nil {
			logrus.Error(err.Error())
		}
	}

}

func destVideo(id string) string {
	return *destDir + "/" + id + ".mp4"
}
func destAudio(id string) string {
	return *destDir + "/" + id + ".mp3"
}

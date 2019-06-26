package ytbrss

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rylio/ytdl"
	"github.com/sirupsen/logrus"
)

type VideoDialog struct {
	DestDir string
}

type Processor struct {
	Title     string
	AudioPath string
	Run       func() error
}

func NewVideoProcessingDialog(destDir string) (*VideoDialog, error) {
	err := os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return &VideoDialog{
		DestDir: destDir,
	}, nil
}

func (d *VideoDialog) Handle(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message) error {
	ctx, cancelFunc := context.WithCancel(ctx)

	url := &url.URL{}
	url, err := url.Parse(msg.Text)
	if err != nil {
		return err
	}

	encodeRes, err := d.GetYoutubeProcessor(url)
	if err != nil {
		return err
	}

	go func() {
		ticker := time.Tick(500 * time.Millisecond)

		progressMessage := tgbotapi.NewMessage(msg.Chat.ID, "File size: 0")
		retMsg, err := bot.Send(progressMessage)
		if err != nil {
			logrus.Error(err)
			return
		}
		var progress int64
		for {

			select {
			case <-ticker:
				curProgress := encodeRes.Progress()
				if  curProgress != progress {
					edit := tgbotapi.NewEditMessageText(msg.Chat.ID, retMsg.MessageID, fmt.Sprintf("File size: %d", curProgress))
					if _, err := bot.Send(edit); err != nil {
						logrus.Error(err)
					}
				}

			case <-ctx.Done():
				edit := tgbotapi.NewEditMessageText(msg.Chat.ID, retMsg.MessageID, "Processing finished. Wait audio")
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
		return err
	}

	sendFile, err := os.OpenFile(encodeRes.AudioPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	reader := tgbotapi.FileReader{
		encodeRes.Title,
		sendFile,
		-1,
	}
	config := tgbotapi.NewAudioUpload(msg.Chat.ID, reader)

	_, err = bot.Send(config)
	if err != nil {
		return err
	}
	return nil
}

func (e *VideoDialog) GetYoutubeProcessor(url *url.URL) (*Processor, error) {
	var info *ytdl.VideoInfo
	var err error
	if info, err = ytdl.GetVideoInfoFromURL(url); err != nil {
		info, err = ytdl.GetVideoInfoFromShortURL(url)
		if err != nil {
			return nil, &UserError{msg: fmt.Sprintf("Not youtube link: %s", url)}
		}
	}

	if _, err := os.Stat(e.destVideo(info.ID)); os.IsNotExist(err) {

		file, err := os.Create(e.destVideo(info.ID))
		if err != nil {
			return nil, err
		}
		err = info.Download(info.Formats[0], file)
		_ = file.Close()
		if err != nil {
			return nil, err
		}
	}

	ret := e.destAudio(info.ID)

	return &Processor{
		Title:     info.Title,
		AudioPath: ret,
		Run: func() error {
			cmd := exec.Command("ffmpeg", "-i", e.destVideo(info.ID),
				"-f", "mp3", "-y", "-ab", "64000",
				"-metadata", fmt.Sprintf("title=\"%s\"", info.Title),
				"-vn", ret)
			err = cmd.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}, nil
}

func (p *Processor) Progress() int64 {
	stat, err := os.Stat(p.AudioPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logrus.Error(err)
		}
		return 0
	}

	return stat.Size()
}

func (e *VideoDialog) destVideo(id string) string {
	return e.DestDir + "/" + id + ".mp4"
}
func (e *VideoDialog) destAudio(id string) string {
	return e.DestDir + "/" + id + ".mp3"
}

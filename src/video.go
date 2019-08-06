package ytbrss

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rylio/ytdl"
	"github.com/sirupsen/logrus"
)

const maxPartDuration = 30*time.Minute
const maxFileSize = 45*1024*1024

type VideoDialog struct {
	DestDir string
}

type Processor struct {
	ID        string
	Title     string
	AudioPath string
	Multifile bool
	Run       func() error
	DestDir   string

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
	audiofiles, err := encodeRes.Audiofiles()
	if err != nil {
		return err
	}
	for i, audioPath := range audiofiles {
		sendFile, err := os.OpenFile(audioPath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}

		var partName string
		if i == 0 {
			partName = encodeRes.Title
		} else {
			partName = fmt.Sprintf("%s. Part %d", encodeRes.Title, i + 1)
		}
		reader := tgbotapi.FileReader{
			partName,
			sendFile,
			-1,
		}
		config := tgbotapi.NewAudioUpload(msg.Chat.ID, reader)

		_, err = bot.Send(config)
		if err != nil {
			return err
		}
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
		err = info.Download(info.Formats[len(info.Formats) - 1], file)
		_ = file.Close()
		if err != nil {
			return nil, err
		}
	}

	parts := info.Duration.Nanoseconds() / maxPartDuration.Nanoseconds()
	audio := e.destAudio(info.ID)

	return &Processor{
		Title:     info.Title,
		AudioPath: audio,
		Multifile: parts > 0,
		DestDir: e.DestDir,
		Run: func() error {

			 cmd := exec.Command("ffmpeg", "-i", e.destVideo(info.ID),
					"-f", "mp3", "-y", "-ab", "64000",
					"-metadata", fmt.Sprintf("title=\"%s\"", info.Title),
					"-vn", audio)
			 err = cmd.Run()
			 if err != nil {
				return err
			 }

			s, err := os.Stat(audio)
			if err != nil {
				return err
			}
			if s.Size() > maxFileSize {
			 	cmd = exec.Command("ffmpeg", "-i", audio, "-f", "segment", "-segment_time", strconv.Itoa(int(maxPartDuration.Seconds())), "-c",
			 		"copy", e.destAudioPart(info.ID))
			 	err = cmd.Run()
			 	if err != nil {
			 		return err
				}
			 }

			 return nil
		},
	}, nil
}

func (p *Processor) Audiofiles() ([]string, error) {
	files, err := ioutil.ReadDir(p.DestDir)
	if err != nil {
		return nil, err
	}
	s, err := os.Stat(p.AudioPath)
	if err != nil {
		return nil, err
	}
	if s.Size() > maxFileSize {
		var res []string
		for _, f := range files {
			if strings.Contains(f.Name(), p.ID) && strings.Contains(f.Name(), "part") {
				res = append(res, p.DestDir + "/" + f.Name())
			}
		}

		return res, nil
	}
	return []string{p.AudioPath}, nil
}

func (p *Processor) Progress() int64 {
	fullSize := int64(0)

	stat, err := os.Stat(p.AudioPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logrus.Error(err)
		}
			return 0
	}
	fullSize += stat.Size()

	return fullSize
}

func (e *VideoDialog) destVideo(id string) string {
	return e.DestDir + "/" + id + ".mp4"
}

func (e *VideoDialog) destAudioPart(id string) string {
	return fmt.Sprintf(`%s/%s_part_%%03d.mp3`, e.DestDir, id)
}

func (e *VideoDialog) destAudio(id string) string {
	return fmt.Sprintf("%s/%s.mp3", e.DestDir, id)
}

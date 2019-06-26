package ytbrss

import (
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/rylio/ytdl"
	"github.com/sirupsen/logrus"
)

type Encoder struct {
	DestDir string
}

type Processor struct {
	Title string
	AudioPath string
	Run func() error

}

func NewEncoder(destDir string) (*Encoder, error) {
	err := os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return &Encoder{
		DestDir: destDir,
	}, nil
}

func (e *Encoder) GetProcessor(url *url.URL) (*Processor, error) {
	info, err := ytdl.GetVideoInfoFromURL(url)
	if err != nil {
		return nil, err
	}
	file, err := os.Create(e.destVideo(info.ID))
	if err != nil {
		return nil, err
	}
	err = info.Download(info.Formats[0], file)
	_ = file.Close()
	if err != nil {
		return nil, err
	}

	ret := e.destAudio(info.ID)

	return &Processor{
		Title: info.Title,
		AudioPath: ret,
		Run: func() error {
			cmd := exec.Command("ffmpeg", "-i", e.destVideo(info.ID), "-f", "mp3", "-y", "-ab", "64000", "-vn", ret)
			err = cmd.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}, nil
}


func (p *Processor) Progress() time.Duration {
	stat, err := os.Stat(p.AudioPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logrus.Error(err)
		}
		return time.Duration(0)
	}

	return time.Duration(stat.Size())
}

func (e *Encoder) destVideo(id string) string {
	return e.DestDir + "/" + id + ".mp4"
}
func (e *Encoder) destAudio(id string) string {
	return e.DestDir + "/" + id + ".mp3"
}


package ytbrss

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"

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

func (e *Encoder) GetYoutubeProcessor(url *url.URL) (*Processor, error) {
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

func (e *Encoder) destVideo(id string) string {
	return e.DestDir + "/" + id + ".mp4"
}
func (e *Encoder) destAudio(id string) string {
	return e.DestDir + "/" + id + ".mp3"
}


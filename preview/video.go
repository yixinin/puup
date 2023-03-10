package preview

import (
	"bytes"
	"fmt"
	"os"

	"github.com/disintegration/imaging"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/yixinin/puup/stderr"
)

// SaveVideoPreview 生成视频缩略图并保存（作为封面）
func SaveVideoPreview(videoPath, snapshotPath string, frameNum int) error {
	buf := bytes.NewBuffer(nil)
	err := ffmpeg.Input(videoPath).
		Filter("select", ffmpeg.Args{fmt.Sprintf("gte(n,%d)", frameNum)}).
		Output("pipe:", ffmpeg.KwArgs{"vframes": 1, "format": "image2", "vcodec": "mjpeg"}).
		WithOutput(buf, os.Stdout).
		Run()
	if err != nil {
		return stderr.Wrap(err)
	}

	img, err := imaging.Decode(buf)
	if err != nil {
		return stderr.Wrap(err)
	}

	err = imaging.Save(img, snapshotPath)
	if err != nil {
		return stderr.Wrap(err)
	}
	return nil
}

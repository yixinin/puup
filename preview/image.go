package preview

import (
	"os"

	"github.com/disintegration/imaging"
	"github.com/yixinin/puup/stderr"
)

func SaveImagePreview(filename, snapshotPath string) error {
	fs, err := os.Open(filename)
	if err != nil {
		return stderr.Wrap(err)
	}
	defer fs.Close()
	image, err := imaging.Decode(fs)
	if err != nil {
		return stderr.Wrap(err)
	}
	var w = image.Bounds().Max.X
	var h = image.Bounds().Max.X
	r := float64(w) / float64(h)
	h = 200
	w = int(float64(h) * r)
	//生成缩略图，尺寸150*200，并保持到为文件2.jpg
	image = imaging.Resize(image, w, h, imaging.Lanczos)
	err = imaging.Save(image, snapshotPath)
	if err != nil {
		return stderr.Wrap(err)
	}
	return nil
}

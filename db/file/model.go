package file

import (
	"fmt"
	"os"
	"time"
)

type File struct {
	Etag        string   `json:"etag"`
	Size        uint64   `json:"size"`
	Path        string   `json:"path"`
	PreviewPath string   `json:"preview"`
	Type        FileType `json:"type"`
	Reference   int      `json:"ref"`
}

func (f *File) Key() string {
	return GetFileKey(f.Etag, f.Size)
}

func GetFileKey(etag string, size uint64) string {
	return fmt.Sprintf("file/%s/%d", etag, size)
}
func GetFileName(etag string, size uint64, ext string) (string, string) {
	var dir, _ = os.Getwd()
	filename := fmt.Sprintf("%s/files/%s_%d.%s", dir, etag, size, ext)
	previewFilename := fmt.Sprintf("%s/previews/%s_%d.png", dir, etag, size)
	return filename, previewFilename
}

func GetUserFileKey(path string) string {
	return fmt.Sprintf("file/user/%s", path)
}

type FileType uint8

const (
	TypeImage = 1
	TypeVideo = 2
	TypeAudio = 3
	TypeDoc   = 4
	TypeOther = 10
)

func (t FileType) String() string {
	switch t {
	case TypeImage:
		return "image"
	case TypeVideo:
		return "video"
	case TypeAudio:
		return "audio"
	case TypeDoc:
		return "doc"
	}
	return "other"
}

type UserFile struct {
	Path        string   `json:"path"`
	Etag        string   `json:"etag"`
	Size        uint64   `json:"size"`
	RealPath    string   `json:"realPath"`
	PreviewPath string   `json:"preview"`
	Type        FileType `json:"type"`
	CreateTime  int64    `json:"create"`
	UpdateTime  int64    `json:"update"`
}

func CopyFile(f File, path string) UserFile {
	var now = time.Now().Unix()
	return UserFile{
		Path:        path,
		Etag:        f.Etag,
		Size:        f.Size,
		RealPath:    f.Path,
		PreviewPath: f.PreviewPath,
		Type:        f.Type,
		CreateTime:  now,
		UpdateTime:  now,
	}
}

func (f *UserFile) Key() string {
	return f.Path
}

type TreeNode struct {
	Path       string        `json:"path"`
	IsDir      bool          `json:"dir"`
	CreateTime int64         `json:"create"`
	UpdateTime int64         `json:"update"`
	Children   TreeNodeSlice `json:"children"`
}

func (d *TreeNode) Key() string {
	return d.Path
}
func (d *TreeNode) AddDir(dir string) {
	d.Children = append(d.Children, NewDirNode(dir))
}
func (d *TreeNode) AddFile(path string) {
	d.Children = append(d.Children, NewFileNode(path))
}

func NewFileNode(path string) *TreeNode {
	var now = time.Now().Unix()
	node := &TreeNode{
		Path:       path,
		IsDir:      false,
		CreateTime: now,
		UpdateTime: now,
	}
	return node
}
func NewDirNode(path string, cap ...int) *TreeNode {
	var now = time.Now().Unix()
	node := &TreeNode{
		Path:       path,
		IsDir:      true,
		CreateTime: now,
		UpdateTime: now,
	}

	if len(cap) == 0 {
		return node
	}
	node.Children = make(TreeNodeSlice, 0, cap[0])
	return node
}

type TreeNodeSlice []*TreeNode

func (a TreeNodeSlice) Len() int           { return len(a) }
func (a TreeNodeSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TreeNodeSlice) Less(i, j int) bool { return a[i].Path < a[j].Path }

package file

import (
	"fmt"
	"time"
)

type File struct {
	Etag      string   `json:"etag"`
	Size      uint64   `json:"size"`
	Path      string   `json:"path"`
	Type      FileType `json:"type"`
	Reference int      `json:"ref"`
}

func (f *File) Key() string {
	return GetFileKey(f.Etag, f.Size)
}

func GetFileKey(etag string, size uint64) string {
	return fmt.Sprintf("file/%s/%d", etag, size)
}

type FileType uint8

const (
	TypeImage = 1
	TypeVideo = 2
	TypeAudio = 3
	TypeDoc   = 4
	TypeOther = 10
)

type UserFile struct {
	Path       string   `json:"path"`
	Etag       string   `json:"etag"`
	Size       uint64   `json:"size"`
	RealPath   string   `json:"realPath"`
	FileType   FileType `json:"type"`
	CreateTime int64    `json:"create"`
	UpdateTime int64    `json:"update"`
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

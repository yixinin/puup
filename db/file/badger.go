package file

import (
	"context"

	"github.com/yixinin/puup/db"
)

type BadgerStorage struct {
}

func init() {
	storage = &BadgerStorage{}
}

func (b *BadgerStorage) InsertFile(ctx context.Context, file File) error {
	file.Reference = 1
	_, _, err := db.GetOrSet(ctx, file.Key(), file, 0)
	return err
}
func (b *BadgerStorage) IncrReference(ctx context.Context, etag string, size uint64, inc int) (int, error) {
	var ref int
	err := db.Update(ctx, GetFileKey(etag, size), func(value *File) error {
		ref = value.Reference + inc
		value.Reference = ref
		return nil
	})
	return ref, err
}
func (b *BadgerStorage) GetFile(ctx context.Context, etag string, size uint64) (File, error) {
	var key = GetFileKey(etag, size)
	return db.Get[File](ctx, key)
}

func (b *BadgerStorage) InsertUserFile(ctx context.Context, file UserFile) error {
	return db.Set(ctx, GetUserFileKey(file.Path), file, 0)
}

func (b *BadgerStorage) GetUserFile(ctx context.Context, path string) (UserFile, error) {
	return db.Get[UserFile](ctx, GetUserFileKey(path))
}

func (b *BadgerStorage) Rename(ctx context.Context, oldPath, newPath string) error {
	return db.Update(ctx, GetUserFileKey(oldPath), func(value *File) error {
		value.Path = newPath
		return nil
	})
}

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
	return db.Set(ctx, file.Key(), file.Path, 0)
}

func (b *BadgerStorage) GetFile(ctx context.Context, etag string, size uint64) (File, error) {
	var key = GetFileKey(etag, size)
	return db.Get[File](ctx, key)
}

package file

import (
	"context"

	"github.com/yixinin/puup/storage"
)

type BadgerStorage struct {
}

func (b *BadgerStorage) InsertFile(ctx context.Context, file File) error {
	return storage.Set(ctx, file.Key(), file.Path, 0)
}

func (b *BadgerStorage) GetFile(ctx context.Context, etag string, size uint64) (File, error) {
	var key = GetFileKey(etag, size)
	return storage.Get[File](ctx, key)
}

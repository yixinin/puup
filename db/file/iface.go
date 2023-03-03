package file

import "context"

type Storage interface {
	InsertFile(ctx context.Context, file File) error
	InsertUserFile(ctx context.Context, file UserFile) error

	GetFile(ctx context.Context, etag string, size uint64) (File, error)
	GetUserFile(ctx context.Context, path string) (UserFile, error)
}

var storage Storage

func GetStorage() Storage {
	return storage
}

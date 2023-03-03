package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type Storage struct {
	db *badger.DB
}

var storage *Storage

func Init() {
	db, err := badger.Open(badger.DefaultOptions("data/db"))
	if err != nil {
		panic(err)
	}
	storage = &Storage{db: db}
}

func Set[T any](ctx context.Context, key string, value T, ttl int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return storage.db.Update(func(txn *badger.Txn) error {
		if ttl <= 0 {
			return txn.Set([]byte(key), data)
		}
		e := badger.NewEntry([]byte(key), data).
			WithTTL(time.Second * time.Duration(ttl))
		return txn.SetEntry(e)
	})
}

func Get[T any](ctx context.Context, key string) (T, error) {
	var t T
	return t, storage.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &t)
		})
	})
}

func Delete(ctx context.Context, key string) error {
	return storage.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func Scan[T any](ctx context.Context, prefix string, limit int) ([]T, error) {
	var ts = make([]T, 0, limit)
	return ts, storage.db.View(func(txn *badger.Txn) error {

		var opt = badger.DefaultIteratorOptions
		if prefix != "" {
			opt.Prefix = []byte(prefix)
		}
		iter := txn.NewIterator(opt)
		var i int
		for iter.Rewind(); iter.Valid() && i < limit; iter.Next() {
			i++
			var t T
			err := iter.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &t)
			})
			if err != nil {
				return err
			}
			ts = append(ts, t)
			if len(ts) == limit {
				return nil
			}
		}
		return nil
	})
}

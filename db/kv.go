package db

import (
	"context"
	"encoding/json"
	"errors"
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
func GetOrSet[T any](ctx context.Context, key string, value T, ttl int) (bool, T, error) {
	var ok bool
	err := storage.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil && errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}
		if item != nil {
			ok = true
			return item.Value(func(val []byte) error {
				return json.Unmarshal(val, &value)
			})
		}

		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if ttl <= 0 {
			return txn.Set([]byte(key), data)
		}
		e := badger.NewEntry([]byte(key), data).
			WithTTL(time.Second * time.Duration(ttl))
		return txn.SetEntry(e)
	})
	return ok, value, err
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
func Update[T any](ctx context.Context, key string, f func(value *T) error) error {
	return storage.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil || item == nil {
			return err
		}
		var value T

		err = item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, &value); err != nil {
				return err
			}
			return f(&value)
		})
		if err != nil {
			return err
		}

		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(key), data)
		if exp := item.ExpiresAt(); exp > 0 {
			var now = time.Now().Unix()
			ttl := exp - uint64(now)
			e = e.WithTTL(time.Second * time.Duration(ttl))
		}
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

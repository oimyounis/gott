package gott

import (
	"github.com/dgraph-io/badger"
	js "github.com/json-iterator/go"
)

type SessionStore struct {
	*badger.DB
}

func LoadSessionStore() (*SessionStore, error) {
	opts := badger.DefaultOptions(".sessions.store")
	opts.EventLogging = false

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &SessionStore{db}, nil
}

func (ss *SessionStore) Get(key string, out *Session) error {
	return ss.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return js.Unmarshal(val, out)
		})
	})
}

func (ss *SessionStore) Exists(key string) bool {
	return ss.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	}) == nil
}

func (ss *SessionStore) Set(key string, value interface{}) error {
	txn := ss.NewTransaction(true)

	if err := set(txn, key, value); err == badger.ErrTxnTooBig {
		err = txn.Commit()
		if err != nil {
			return err
		}

		txn = ss.NewTransaction(true)
		err := set(txn, key, value)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	err := txn.Commit()

	if err != nil {
		return err
	}

	return nil
}

func (ss *SessionStore) Delete(key string) error {
	return ss.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func set(txn *badger.Txn, key string, value interface{}) error {
	valJson, err := js.Marshal(value)
	if err != nil {
		return nil
	}

	return txn.Set([]byte(key), valJson)
}

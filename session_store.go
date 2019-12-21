package gott

import (
	"github.com/dgraph-io/badger"
	js "github.com/json-iterator/go"
)

type Session struct {
	client       *Client
	MessageStore *MessageStore
	clean        bool
}

func NewSession(client *Client, cleanFlag string) *Session {
	return &Session{
		client:       client,
		MessageStore: NewMessageStore(),
		clean:        cleanFlag == "1",
	}
}

func (s *Session) Get() (map[string]interface{}, error) {
	return GOTT.SessionStore.Get(s.client.ClientId)
}

func (s *Session) Update(value map[string]interface{}) error {
	return GOTT.SessionStore.Set(s.client.ClientId, value)
}

func (s *Session) Put() error {
	return GOTT.SessionStore.Set(s.client.ClientId, s)
}

func (s *Session) Client() *Client {
	return s.client
}

func (s *Session) Clean() bool {
	return s.clean
}

//func (s *Session) Put() error {
//	return GOTT.SessionStore.Set(s.Client.ClientId, s.MessageStore)
//}

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

func (ss *SessionStore) Get(key string) (map[string]interface{}, error) {
	data := map[string]interface{}{}

	err := ss.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			err := js.Unmarshal(val, &data)

			return err
		})

		return err
	})

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (ss *SessionStore) Exists(key string) bool {
	err := ss.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})

	if err != nil {
		return false
	}

	return true
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

	err = txn.Set([]byte(key), valJson)

	return err
}

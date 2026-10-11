// Package storage provides ACID transactions using bbolt, a pure-Go embedded database.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MMH-4ALL/vantage-cli/internal/domain"
	bolt "go.etcd.io/bbolt"
)

type Repository interface {
	Read() (Snapshot, error)
	Update(func(*Snapshot) error) error
	Close() error
}

type Snapshot struct {
	Schema    int               `json:"schema"`
	Projects  []domain.Project  `json:"projects"`
	Tasks     []domain.Task     `json:"tasks"`
	Workflows []domain.Workflow `json:"workflows"`
	State     domain.State      `json:"state"`
}

type Store struct {
	path   string
	mu     sync.Mutex
	closed bool
}

var bucket = []byte("vantage")
var key = []byte("state")

func Open(path string) (*Store, error) {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, e
	}
	db, e := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if e != nil {
		return nil, fmt.Errorf("open local database (another VANTAGE process may be using it): %w", e)
	}
	s := &Store{path: path}
	e = db.Update(func(tx *bolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists(bucket)
		if e != nil {
			return e
		}
		if b.Get(key) == nil {
			raw, e := json.Marshal(Snapshot{Schema: 1, Projects: []domain.Project{}, Tasks: []domain.Task{}, Workflows: []domain.Workflow{}})
			if e != nil {
				return e
			}
			return b.Put(key, raw)
		}
		_, e = decode(b.Get(key))
		return e
	})
	if e != nil {
		db.Close()
		return nil, e
	}
	if e = db.Close(); e != nil {
		return nil, e
	}
	return s, nil
}

func decode(b []byte) (Snapshot, error) {
	var s Snapshot
	if e := json.Unmarshal(b, &s); e != nil {
		return s, fmt.Errorf("database data is damaged; preserve the file before repair: %w", e)
	}
	if s.Schema != 1 {
		return s, errors.New("unsupported database version; upgrade VANTAGE")
	}
	return s, nil
}

// Each transaction owns its file lock briefly. An open dashboard does not lock
// other CLI processes out for the lifetime of the session.
func (s *Store) transact(write bool, fn func(*bolt.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("database handle is closed")
	}
	if _, e := os.Stat(s.path); e != nil {
		return fmt.Errorf("database file is unavailable: %w", e)
	}
	db, e := bolt.Open(s.path, 0600, &bolt.Options{Timeout: time.Second})
	if e != nil {
		return fmt.Errorf("database is busy or unavailable: %w", e)
	}
	defer db.Close()
	if write {
		return db.Update(fn)
	}
	return db.View(fn)
}
func (s *Store) Read() (Snapshot, error) {
	var out Snapshot
	e := s.transact(false, func(tx *bolt.Tx) error {
		var e error
		b := tx.Bucket(bucket)
		if b == nil {
			return errors.New("database schema is missing; restore your backup")
		}
		out, e = decode(b.Get(key))
		return e
	})
	return out, e
}
func (s *Store) Update(fn func(*Snapshot) error) error {
	return s.transact(true, func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return errors.New("database schema is missing; restore your backup")
		}
		state, e := decode(b.Get(key))
		if e != nil {
			return e
		}
		if e = fn(&state); e != nil {
			return e
		}
		raw, e := json.Marshal(state)
		if e != nil {
			return e
		}
		return b.Put(key, raw)
	})
}
func (s *Store) Close() error { s.mu.Lock(); defer s.mu.Unlock(); s.closed = true; return nil }

package storage

import (
	"fmt"
	"go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"runtime"
)

type (
	Storage interface {
		Init() error
		Set(key, value string) error
		Get(key string) (string, error)
		Close()
	}
	storage struct {
		boldDBName string
		db         *bbolt.DB
	}
)

func NewStorage() Storage {
	return &storage{
		boldDBName: "synexis-cli-cache.db",
	}
}

// platform-specific default path
func getDefaultDBPath(fileName string) (string, error) {
	var basePath string
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		basePath = filepath.Join(home, "Library", "Caches", "synexis-cli")
	case "linux":
		basePath = filepath.Join(home, ".cache", "synexis-cli")
	case "windows":
		appData := os.Getenv("AppData")
		if appData == "" {
			return "", fmt.Errorf("AppData environment variable not set")
		}
		basePath = filepath.Join(appData, "synexis-cli")
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Ensure directory exists
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return "", err
	}

	return filepath.Join(basePath, fileName), nil
}

func (s *storage) Init() error {
	dbPath, err := getDefaultDBPath(s.boldDBName)
	if err != nil {
		return err
	}

	s.db, err = bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(s.boldDBName))
		return err
	})
}

func (s *storage) Set(key, value string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(s.boldDBName))
		return b.Put([]byte(key), []byte(value))
	})
}

func (s *storage) Get(key string) (string, error) {
	var val string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(s.boldDBName))
		v := b.Get([]byte(key))
		if v != nil {
			val = string(v)
		}
		return nil
	})
	return val, err
}

func (s *storage) Close() {
	if s.db != nil {
		_ = s.db.Close()
	}
}

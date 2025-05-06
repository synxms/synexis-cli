package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
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

type (
	Authentication interface {
		GenerateLoginWithGoogle() (*LoginResponse, error)
		OpenDefaultBrowser(url string) error
	}
	authentication struct {
		loginEndpoint         string
		contentTypeJsonHeader string
	}
	LoginResponse struct {
		ResponseCode    string `json:"responseCode"`
		ResponseMessage string `json:"responseMessage"`
		RedirectURL     string `json:"redirectUrl"`
	}
)

func NewAuthentication() Authentication {
	return &authentication{
		contentTypeJsonHeader: "application/json",
		loginEndpoint:         "http://localhost:2343/api/v1/authentication/login",
	}
}

func (a *authentication) GenerateLoginWithGoogle() (*LoginResponse, error) {
	resp, err := http.Post(a.loginEndpoint, a.contentTypeJsonHeader, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, errors.New("failed to contact authentication server")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic("failed to contact authentication server")
		}
	}(resp.Body)
	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, errors.New("failed to contact authentication server")
	}
	return &loginResp, nil
}

func (a *authentication) OpenDefaultBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

var (
	rootCmd = &cobra.Command{
		Use:   "synexis",
		Short: "Authentication tools for synexis",
		Long:  `Authentication tools for synexis`,
	}
	apiKeyCmd = &cobra.Command{
		Use:   "apikey",
		Short: "Generate api key for accessing synexis product",
		Long:  `Generate api key for accessing synexis product`,
	}
	tokenCmd = &cobra.Command{
		Use:   "token",
		Short: "Token management after authentication",
		Long:  `Token management after authentication`,
	}
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "authenticate",
		Short: "Authentication to create synexis account",
		Long:  `Authentication to create synexis account`,
		RunE: func(cmd *cobra.Command, args []string) error {
			authenticationService := NewAuthentication()
			result, err := authenticationService.GenerateLoginWithGoogle()
			if err != nil {
				log.Fatalln(err.Error())
			}
			if result != nil {
				err := authenticationService.OpenDefaultBrowser(result.RedirectURL)
				if err != nil {
					log.Fatalln(err.Error())
				}
			}
			return nil
		},
	})
	rootCmd.AddCommand(apiKeyCmd)
	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "sentinel",
		Short: "Generate api key for accessing sentinel model",
		Long:  `Generate api key for accessing sentinel model`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "fblob",
		Short: "Generate api key for accessing synexis file bucket",
		Long:  `Generate api key for accessing synexis file bucket`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "set-access [token]",
		Short: "Set new access token to local synexis command line tool",
		Long:  `Set new access token to local synexis command line tool`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage := NewStorage()
			if err := storage.Init(); err != nil {
				log.Fatalln("Failed to init storage:", err)
			}
			defer storage.Close()
			if err := storage.Set("access_token", args[0]); err != nil {
				log.Fatalln("Failed to store access token:", err)
			}
			fmt.Println("Access token saved.")
			return nil
		},
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "set-refresh [token]",
		Short: "Set new refresh token to local synexis command line tool",
		Long:  `Set new refresh token to local synexis command line tool`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage := NewStorage()
			if err := storage.Init(); err != nil {
				log.Fatalln("Failed to init storage:", err)
			}
			defer storage.Close()
			if err := storage.Set("refresh_token", args[0]); err != nil {
				log.Fatalln("Failed to store refresh token:", err)
			}
			fmt.Println("Refresh token saved.")
			return nil
		},
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "get-access [token]",
		Short: "Get existing access token to local synexis command line tool",
		Long:  `Get existing access token to local synexis command line tool`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage := NewStorage()
			if err := storage.Init(); err != nil {
				log.Fatalln("Failed to init storage:", err)
			}
			defer storage.Close()
			result, err := storage.Get("access_token")
			if err != nil {
				log.Fatalln("Failed to store access token:", err)
			}
			fmt.Println(result)
			return nil
		},
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "get-refresh [token]",
		Short: "Get existing refresh token to local synexis command line tool",
		Long:  `Get existing refresh token to local synexis command line tool`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage := NewStorage()
			if err := storage.Init(); err != nil {
				log.Fatalln("Failed to init storage:", err)
			}
			defer storage.Close()
			result, err := storage.Get("refresh_token")
			if err != nil {
				log.Fatalln("Failed to store access token:", err)
			}
			fmt.Println(result)
			return nil
		},
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "refresh",
		Short: "Refresh access token and refresh token",
		Long:  `Refresh access token and refresh token`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage := NewStorage()
			if err := storage.Init(); err != nil {
				log.Fatalln("Failed to init storage:", err)
			}
			defer storage.Close()
			result, err := storage.Get("refresh_token")
			if err != nil {
				log.Fatalln("Failed to store access token:", err)
			}
			fmt.Println(result)
			return nil
		},
	})
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

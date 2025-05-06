package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	guuid "github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
func RandomStringUpperCase(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

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
		GenerateAccessAndRefreshToken(refresh string) (*ResponseRefresh, error)
		OpenDefaultBrowser(url string) error
		IsExpired(jwtString string) (*string, *string, error)
	}
	authentication struct {
		loginEndpoint         string
		refreshEndpoint       string
		contentTypeJsonHeader string
	}
	LoginResponse struct {
		ResponseCode    string `json:"responseCode"`
		ResponseMessage string `json:"responseMessage"`
		RedirectURL     string `json:"redirectUrl"`
	}
	ResponseRefresh struct {
		ResponseCode    string `json:"responseCode"`
		ResponseMessage string `json:"responseMessage"`
		Refresh         string `json:"refresh"`
		Access          string `json:"access"`
	}
)

func NewAuthentication() Authentication {
	const baseUrl = "http://localhost:2343"
	return &authentication{
		contentTypeJsonHeader: "application/json",
		loginEndpoint:         fmt.Sprintf("%s/api/v1/authentication/login", baseUrl),
		refreshEndpoint:       fmt.Sprintf("%s/api/v1/authentication/refresh", baseUrl),
	}
}

func (a *authentication) GenerateLoginWithGoogle() (*LoginResponse, error) {
	req, err := http.NewRequest("POST", a.loginEndpoint, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, errors.New("failed to create request")
	}
	req.Header.Set("Content-Type", a.contentTypeJsonHeader)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("failed to contact server")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic("failed to close response body")
		}
	}(resp.Body)
	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, errors.New("failed to contact server")
	}
	return &loginResp, nil
}

func (a *authentication) GenerateAccessAndRefreshToken(refresh string) (*ResponseRefresh, error) {
	req, err := http.NewRequest("POST", a.refreshEndpoint, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, errors.New("failed to create request")
	}
	req.Header.Set("Content-Type", a.contentTypeJsonHeader)
	req.Header.Set("Authorization", "Bearer "+refresh)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("failed to contact server")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic("failed to close response body")
		}
	}(resp.Body)
	var refreshResp ResponseRefresh
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return nil, errors.New("failed to contact server")
	}
	return &refreshResp, nil
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

func (a *authentication) IsExpired(jwtString string) (*string, *string, error) {
	parser := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(jwtString, claims)
	if err != nil {
		return nil, nil, errors.New("invalid token")
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, nil, errors.New("no expiration field in token")
	}
	expTime := time.Unix(int64(exp), 0)
	now := time.Now()
	expiredAt := expTime.Format(time.DateTime)
	totalRemains := expTime.Sub(now).String()
	if now.After(expTime) {
		return &totalRemains, &expiredAt, errors.New("token is expired")
	}
	return &totalRemains, &expiredAt, nil
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
	rand.Seed(time.Now().UnixNano())
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
				if result.ResponseCode == "00" {
					err := authenticationService.OpenDefaultBrowser(result.RedirectURL)
					if err != nil {
						log.Fatalln(err.Error())
					}
				} else {
					fmt.Println("Authentication failed.")
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
			// SYX[3 alphabet uppercase]-[5 random]-[10 random unique string]-[uuid without strips]
			const apiKeyFormat = "SYX%s-%s-%s-%s"
			upperCaseRandom := RandomStringUpperCase(3)
			fiveUniqueRandom := RandomString(5)
			tenUniqueRandom := RandomString(10)
			uuidWithoutStrip := strings.Replace(guuid.NewString(), "-", "", -1)
			fmt.Println(fmt.Sprintf(apiKeyFormat, upperCaseRandom, fiveUniqueRandom, tenUniqueRandom, uuidWithoutStrip))
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
		Use:   "get-access",
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
		Use:   "get-refresh",
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
			refreshToken, err := storage.Get("refresh_token")
			if err != nil {
				log.Fatalln("Failed to store access token:", err)
			}
			authenticationService := NewAuthentication()
			result, err := authenticationService.GenerateAccessAndRefreshToken(refreshToken)
			if err != nil {
				log.Fatalln(err.Error())
			}
			if result != nil {
				if result.ResponseCode == "00" {
					if err := storage.Set("refresh_token", result.Refresh); err != nil {
						log.Fatalln("Failed to store refresh token:", err)
					}
					if err := storage.Set("access_token", result.Access); err != nil {
						log.Fatalln("Failed to store access token:", err)
					}
					fmt.Println("Renewed Refresh token saved.")
					fmt.Println("Renewed Access token saved.")
				} else {
					fmt.Println("Refresh Token failed.")
				}
			}
			return nil
		},
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "expired-check",
		Short: "Refresh access token and Refresh token expired check",
		Long:  `Refresh access token and Refresh token expired check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage := NewStorage()
			if err := storage.Init(); err != nil {
				log.Fatalln("Failed to init storage:", err)
			}
			defer storage.Close()
			refreshToken, err := storage.Get("refresh_token")
			if err != nil {
				log.Fatalln("Failed to store refresh token:", err)
			}
			accessToken, err := storage.Get("access_token")
			if err != nil {
				log.Fatalln("Failed to store access token:", err)
			}
			authenticationService := NewAuthentication()
			refreshTokenRemaining, refreshTokenExpiredAt, err := authenticationService.IsExpired(refreshToken)
			if err != nil {
				if errors.Is(err, errors.New("token is expired")) {
					fmt.Println("Refresh token expired")
				}
				if !errors.Is(err, errors.New("token is expired")) {
					fmt.Println("Refresh token checking error")
				}
			}
			if refreshTokenRemaining != nil && refreshTokenExpiredAt != nil {
				fmt.Println("Refresh token remaining: " + *refreshTokenRemaining)
				fmt.Println("Refresh token expired at: " + *refreshTokenExpiredAt)
			}
			accessTokenRemaining, accessTokenExpiredAt, err := authenticationService.IsExpired(accessToken)
			if err != nil {
				if errors.Is(err, errors.New("token is expired")) {
					fmt.Println("Access token expired")
				}
				if !errors.Is(err, errors.New("token is expired")) {
					fmt.Println("Access token checking error")
				}
			}
			if accessTokenRemaining != nil && accessTokenExpiredAt != nil {
				fmt.Println("Access token remaining: " + *accessTokenRemaining)
				fmt.Println("Access token expired at: " + *accessTokenExpiredAt)
			}
			return nil
		},
	})
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

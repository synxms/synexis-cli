package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/synexism/synexis/pkg/utility"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type (
	Authentication interface {
		GenerateLoginWithGoogle() (*LoginResponse, error)
		GenerateAccessAndRefreshToken(refresh string) (*ResponseRefresh, error)
		GenerateAPIKeySentinel(prefix, validationLayerOne, validationLayerTwo, access string) (*ResponseRefresh, error)
		UploadFileDatasetSentinel(absoluteFile string) (*ResponseUploadDataset, error)
		OpenDefaultBrowser(url string) error
		IsExpired(jwtString string) (*string, *string, error)
	}
	authentication struct {
		loginEndpoint             string
		refreshEndpoint           string
		generateAPIKeyEndpoint    string
		uploadDatasetFileEndpoint string
		contentTypeJsonHeader     string
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
	ResponseUploadDataset struct {
		ResponseCode    string `json:"success"`
		ResponseMessage string `json:"messages"`
		Data            struct {
			DatasetID string `json:"dataset_id"`
		} `json:"data"`
	}
)

func NewAuthentication(baseUrl string) Authentication {
	if !utility.IsValidURL(baseUrl) {
		log.Fatalln("please provide base url before continue")
	}
	return &authentication{
		contentTypeJsonHeader:     "application/json",
		loginEndpoint:             fmt.Sprintf("%s/api/v1/authentication/login", baseUrl),
		refreshEndpoint:           fmt.Sprintf("%s/api/v1/authentication/refresh", baseUrl),
		generateAPIKeyEndpoint:    fmt.Sprintf("%s/api/v1/authentication/create/apikey", baseUrl),
		uploadDatasetFileEndpoint: fmt.Sprintf("%s/api/v1/sentinel/sessions/upload/dataset", baseUrl),
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

func (a *authentication) UploadFileDatasetSentinel(absoluteFile string) (*ResponseUploadDataset, error) {
	file, err := os.Open(absoluteFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Prepare multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	formFile, err := writer.CreateFormFile("file", filepath.Base(absoluteFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(formFile, file); err != nil {
		return nil, fmt.Errorf("failed to write file to form: %w", err)
	}

	// Close writer to set the terminating boundary
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create POST request
	req, err := http.NewRequest("POST", a.uploadDatasetFileEndpoint, &buf)
	if err != nil {
		return nil, errors.New("failed to create request")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("failed to contact server")
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			panic("failed to close response body")
		}
	}(resp.Body)

	// Parse response
	var uploadResp ResponseUploadDataset
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, errors.New("failed to decode upload response")
	}

	return &uploadResp, nil
}

func (a *authentication) GenerateAPIKeySentinel(prefix, validationLayerOne, validationLayerTwo, access string) (*ResponseRefresh, error) {
	dataRequest := map[string]interface{}{}
	dataRequest["prefix"] = prefix
	dataRequest["validationLayerOne"] = validationLayerOne
	dataRequest["validationLayerTwo"] = validationLayerTwo
	dataRequestBytes, err := json.Marshal(dataRequest)
	if err != nil {
		return nil, errors.New("failed to marshal request")
	}
	req, err := http.NewRequest("POST", a.generateAPIKeyEndpoint, bytes.NewBuffer(dataRequestBytes))
	if err != nil {
		return nil, errors.New("failed to create request")
	}
	req.Header.Set("Content-Type", a.contentTypeJsonHeader)
	req.Header.Set("Authorization", "Bearer "+access)
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

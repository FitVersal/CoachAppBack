package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type StorageService interface {
	UploadFile(ctx context.Context, file multipart.File, filename string, folder string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
	GetSignedURL(ctx context.Context, fileURL string) (string, error)
}

type SupabaseStorageService struct {
	baseURL    string
	bucket     string
	serviceKey string
	httpClient *http.Client
}

func NewSupabaseStorageService(baseURL, bucket, serviceKey string) *SupabaseStorageService {
	return &SupabaseStorageService{
		baseURL:    strings.TrimRight(baseURL, "/"),
		bucket:     bucket,
		serviceKey: serviceKey,
		httpClient: http.DefaultClient,
	}
}

func (s *SupabaseStorageService) UploadFile(ctx context.Context, file multipart.File, filename string, folder string) (string, error) {
	objectPath := path.Join(strings.Trim(folder, "/"), filename)
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, s.bucket, objectPath)

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read upload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("build upload request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("x-upsert", "true")
	req.Header.Set("Content-Type", http.DetectContentType(content))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("upload file: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.baseURL, s.bucket, objectPath), nil
}

func (s *SupabaseStorageService) DeleteFile(ctx context.Context, fileURL string) error {
	objectPath, err := s.objectPathFromURL(fileURL)
	if err != nil {
		return err
	}

	deleteURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, s.bucket, objectPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return fmt.Errorf("build delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("delete file: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (s *SupabaseStorageService) GetSignedURL(ctx context.Context, fileURL string) (string, error) {
	objectPath, err := s.objectPathFromURL(fileURL)
	if err != nil {
		return "", err
	}

	signURL := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", s.baseURL, s.bucket, objectPath)
	payload := map[string]int{"expiresIn": 3600}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal signed url payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, signURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build signed url request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get signed url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("get signed url: status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var response struct {
		SignedURL string `json:"signedURL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode signed url response: %w", err)
	}
	if response.SignedURL == "" {
		return "", fmt.Errorf("signed url missing from response")
	}

	return fmt.Sprintf("%s/storage/v1%s", s.baseURL, response.SignedURL), nil
}

func (s *SupabaseStorageService) objectPathFromURL(fileURL string) (string, error) {
	parsed, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("parse file url: %w", err)
	}

	publicPrefix := "/storage/v1/object/public/" + s.bucket + "/"
	objectPrefix := "/storage/v1/object/" + s.bucket + "/"

	switch {
	case strings.HasPrefix(parsed.Path, publicPrefix):
		return strings.TrimPrefix(parsed.Path, publicPrefix), nil
	case strings.HasPrefix(parsed.Path, objectPrefix):
		return strings.TrimPrefix(parsed.Path, objectPrefix), nil
	default:
		return "", fmt.Errorf("file url does not belong to configured bucket")
	}
}

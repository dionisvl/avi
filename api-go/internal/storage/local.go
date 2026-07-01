package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const LocalEndpointPrefix = "file://"

func IsLocalEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, LocalEndpointPrefix)
}

func LocalRootFromEndpoint(endpoint string) string {
	root := strings.TrimPrefix(endpoint, LocalEndpointPrefix)
	if root == "" {
		return ".uploads"
	}
	return root
}

type LocalStorage struct {
	root string
}

func NewLocalStorage(root string) (*LocalStorage, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create local storage root: %w", err)
	}
	return &LocalStorage{root: root}, nil
}

func (s *LocalStorage) Upload(ctx context.Context, key, _ string, body io.Reader, _ int64) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	rel, err := safeObjectPath(key)
	if err != nil {
		return "", err
	}

	target := filepath.Join(s.root, rel)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("create object directory: %w", err)
	}

	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("create object file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, body); err != nil {
		return "", fmt.Errorf("write object file: %w", err)
	}

	return key, nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	rel, err := safeObjectPath(key)
	if err != nil {
		return err
	}

	if err := os.Remove(filepath.Join(s.root, rel)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete object file: %w", err)
	}
	return nil
}

func safeObjectPath(key string) (string, error) {
	rel := filepath.Clean(strings.TrimPrefix(key, "/"))
	if rel == "." || rel == "" || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid object key: %s", key)
	}
	return rel, nil
}

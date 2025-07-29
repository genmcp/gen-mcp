package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache dir: %w", err)
	}

	dir := filepath.Join(base, ".automcp")

	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to ensure %s exists: %w", dir, err)
	}

	return dir, nil
}

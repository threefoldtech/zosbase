package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// GetVmLogsHandler returns VM logs
func (a *API) GetVmLogsHandler(ctx context.Context, fileName string) (string, error) {
	rootPath := "/var/cache/modules/vmd/logs/"
	fullPath := filepath.Join(rootPath, fileName)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file, path: %s, %w", fullPath, err)
	}

	return string(content), nil
}

package zosapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func (g *ZosAPI) getVmLogsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var fileName string
	if err := json.Unmarshal(payload, &fileName); err != nil {
		return nil, fmt.Errorf("failed to decode file name, expecting string: %w", err)
	}
	rootPath := "/var/cache/modules/vmd/"
	fullPath := filepath.Join(rootPath, fileName)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file, path: %s, %w", fullPath, err)
	}
	return string(content), nil

}

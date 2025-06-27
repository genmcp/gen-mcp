package mcpfile

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
)

func ParseMCPFile(path string) (*MCPFile, error) {
	mcpFile := &MCPFile{}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path to mcpfile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mcpfile: %v", err)
	}

	err = yaml.Unmarshal(data, mcpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal mcpfile: %v", err)
	}

	return mcpFile, nil
}

func (t *Tool) UnmarshalJSON(data []byte) error {
	type Doppleganger Tool

	tmp := struct {
		URL string `json:"url"`
		*Doppleganger
	}{
		Doppleganger: (*Doppleganger)(t),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	parsedUrl, err := url.Parse(tmp.URL)
	if err != nil {
		return err
	}
	t.URL = *parsedUrl

	return nil
}

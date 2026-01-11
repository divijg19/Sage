package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type userConfig struct {
	Editor string   `json:"editor,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

func configPath() string {
	dir := sageDir()
	if dir == "" {
		return ""
	}
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "config.json")
}

func loadConfig() (userConfig, error) {
	path := configPath()
	if path == "" {
		return userConfig{}, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return userConfig{}, nil
		}
		return userConfig{}, err
	}

	var cfg userConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return userConfig{}, err
	}

	return cfg, nil
}

func saveConfig(cfg userConfig) error {
	path := configPath()
	if path == "" {
		return nil
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0o600)
}

func getConfiguredEditor() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	return cfg.Editor, nil
}

func setConfiguredEditor(editor string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.Editor = editor
	return saveConfig(cfg)
}

func unsetConfiguredEditor() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.Editor = ""
	return saveConfig(cfg)
}

func getConfiguredTags() ([]string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return append([]string(nil), cfg.Tags...), nil
}

func setConfiguredTags(tags []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.Tags = append([]string(nil), tags...)
	return saveConfig(cfg)
}

func ensureTagsConfigured(tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	current, err := getConfiguredTags()
	if err != nil {
		return err
	}

	set := make(map[string]struct{}, len(current))
	for _, t := range current {
		set[t] = struct{}{}
	}

	changed := false
	for _, t := range tags {
		if t == "" {
			continue
		}
		if _, ok := set[t]; ok {
			continue
		}
		current = append(current, t)
		set[t] = struct{}{}
		changed = true
	}

	if !changed {
		return nil
	}
	return setConfiguredTags(current)
}

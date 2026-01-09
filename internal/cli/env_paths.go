package cli

import (
	"os"
	"path/filepath"
)

func sageDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".sage")
}

func templateDir() string {
	dir := filepath.Join(sageDir(), "templates")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import (
	"os"
	"path/filepath"
	"strings"
)

const SHAFilename = ".farseek_sha"

// ReadSHA reads the last analyzed commit SHA from the .farseek_sha file.
func ReadSHA(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, SHAFilename))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteSHA updates the .farseek_sha file with the given commit SHA.
func WriteSHA(dir, sha string) error {
	return os.WriteFile(filepath.Join(dir, SHAFilename), []byte(sha), 0644)
}

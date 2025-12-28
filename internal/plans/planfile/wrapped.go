// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package planfile

import (
	"errors"
	"fmt"

	"github.com/rafagsiqueira/farseek/internal/encryption"
)

// WrappedPlanFile is a sum type that represents a saved plan, loaded from a
// file path passed on the command line. If the specified file was a thick local
// plan file, the Local field will be populated; if it was a bookmark for a
// remote cloud plan, the Cloud field will be populated. In both cases, the
// other field is expected to be nil. Finally, the outer struct is also expected
// to be used as a pointer, so that a nil value can represent the absence of any
// plan file.
type WrappedPlanFile struct {
	local *Reader
}

func (w *WrappedPlanFile) IsLocal() bool {
	return w != nil && w.local != nil
}

// Local checks whether the wrapped value is a local plan file, and returns it if available.
func (w *WrappedPlanFile) Local() (*Reader, bool) {
	if w != nil && w.local != nil {
		return w.local, true
	} else {
		return nil, false
	}
}

// NewWrappedLocal constructs a WrappedPlanFile from an already loaded local
// plan file reader. Most cases should use OpenWrapped to load from disk
// instead. If the provided reader is nil, the returned pointer is nil.
func NewWrappedLocal(l *Reader) *WrappedPlanFile {
	if l != nil {
		return &WrappedPlanFile{local: l}
	} else {
		return nil
	}
}

// OpenWrapped loads a local or cloud plan file from a specified file path, or
// returns an error if the file doesn't seem to be a plan file of either kind.
// Most consumers should use this and switch behaviors based on the kind of plan
// they expected, rather than directly using Open.
func OpenWrapped(filename string, enc encryption.PlanEncryption) (*WrappedPlanFile, error) {
	// First, try to load it as a local planfile.
	local, localErr := Open(filename, enc)
	if localErr == nil {
		return &WrappedPlanFile{local: local}, nil
	}

	// If neither worked, prioritize definitive "confirmed the format but can't
	// use it" errors, then fall back to dumping everything we know.
	var ulp *ErrUnusableLocalPlan
	if errors.As(localErr, &ulp) {
		return nil, ulp
	}

	return nil, fmt.Errorf("couldn't load the provided path as a local plan file: %w", localErr)
}

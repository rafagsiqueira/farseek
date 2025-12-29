// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statefile

import (
	"io"

	"github.com/rafagsiqueira/farseek/internal/encryption"
	tfversion "github.com/rafagsiqueira/farseek/version"
)

// Write writes the given state to the given writer in the current state
// serialization format.
func Write(s *File, w io.Writer, enc encryption.StateEncryption) error {
	// Always record the current farseek version in the state.
	s.TerraformVersion = tfversion.SemVer

	diags := writeStateV4(s, w, enc)
	return diags.Err()
}

// WriteForTest writes the given state to the given writer in the current state
// serialization format without recording the current farseek version. This is
// intended for use in tests that need to override the current farseek
// version.
func WriteForTest(s *File, w io.Writer) error {
	diags := writeStateV4(s, w, encryption.StateEncryptionDisabled())
	return diags.Err()
}

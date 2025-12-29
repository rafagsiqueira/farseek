// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package aesgcm_test

import (
	"testing"

	"github.com/rafagsiqueira/farseek/internal/encryption/method/aesgcm"
)

func TestDescriptor(t *testing.T) {
	if id := aesgcm.New().ID(); id != "aes_gcm" {
		t.Fatalf("Incorrect descriptor ID returned: %s", id)
	}
}

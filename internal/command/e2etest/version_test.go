// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package e2etest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafagsiqueira/farseek/internal/e2e"
	"github.com/rafagsiqueira/farseek/version"
)

func TestVersion(t *testing.T) {
	// Along with testing the "version" command in particular, this serves
	// as a good smoke test for whether the Farseek binary can even be
	// compiled and run, since it doesn't require any external network access
	// to do its job.

	t.Parallel()

	fixturePath := filepath.Join("testdata", "empty")
	tf := e2e.NewBinary(t, farseekBin, fixturePath)

	stdout, stderr, err := tf.Run("version")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	wantVersion := fmt.Sprintf("Farseek v%s", version.String())
	if !strings.Contains(stdout, wantVersion) {
		t.Errorf("output does not contain our current version %q:\n%s", wantVersion, stdout)
	}
}

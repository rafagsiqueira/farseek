// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tracing

import "testing"

func TestExtractImportPath(t *testing.T) {
	tests := []struct {
		fullName string
		expected string
	}{
		{
			fullName: "github.com/rafagsiqueira/farseek/internal/getproviders.(*registryClient).Get",
			expected: "github.com/rafagsiqueira/farseek/internal/getproviders",
		},
		{
			fullName: "github.com/rafagsiqueira/farseek/pkg/module.Function",
			expected: "github.com/rafagsiqueira/farseek/pkg/module",
		},
		{
			fullName: "main.main",
			expected: "main",
		},
		{
			fullName: "unknownFormat",
			expected: "unknown",
		},
	}

	for _, test := range tests {
		got := extractImportPath(test.fullName)
		if got != test.expected {
			t.Errorf("extractImportPath(%q) = %q; want %q", test.fullName, got, test.expected)
		}
	}
}

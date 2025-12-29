// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import (
	"testing"
)

func TestNullGraphWalker_impl(t *testing.T) {
	var _ GraphWalker = NullGraphWalker{}
}

// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import (
	"testing"
)

func TestStopHook_impl(t *testing.T) {
	var _ Hook = new(stopHook)
}

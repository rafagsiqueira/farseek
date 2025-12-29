// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import (
	"testing"
)

func TestCallbackUIOutput_impl(t *testing.T) {
	var _ UIOutput = new(CallbackUIOutput)
}

// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"os"
	"syscall"
)

var ignoreSignals = []os.Signal{os.Interrupt}
var forwardSignals = []os.Signal{syscall.SIGTERM}

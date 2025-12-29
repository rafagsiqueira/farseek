//go:build !unix
// +build !unix

// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"github.com/rafagsiqueira/farseek/internal/command/webbrowser"
)

func browserLauncherFromEnv() webbrowser.Launcher {
	// We know of no environment variable convention for the current platform.
	return nil
}

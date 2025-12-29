// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/rafagsiqueira/farseek/internal/encryption"
	"github.com/rafagsiqueira/farseek/internal/states/statemgr"
)

// Attempt to open and lock a farseek state file.
// Lock failure exits with 0 and writes "lock failed" to stderr.
func main() {
	if len(os.Args) != 2 {
		log.Fatal(os.Args[0], "statefile")
	}

	s := statemgr.NewFilesystem(os.Args[1], encryption.StateEncryptionDisabled())

	info := statemgr.NewLockInfo()
	info.Operation = "test"
	info.Info = "state locker"

	_, err := s.Lock(context.Background(), info)
	if err != nil {
		io.WriteString(os.Stderr, "lock failed")
	}
}

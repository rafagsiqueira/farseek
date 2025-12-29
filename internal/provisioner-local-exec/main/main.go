//go:build ignore

// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	localexec "github.com/rafagsiqueira/farseek/internal/builtin/provisioners/local-exec"
	"github.com/rafagsiqueira/farseek/internal/grpcwrap"
	"github.com/rafagsiqueira/farseek/internal/plugin"
	"github.com/rafagsiqueira/farseek/internal/tfplugin5"
)

func main() {
	// Provide a binary version of the internal terraform provider for testing
	plugin.Serve(&plugin.ServeOpts{
		GRPCProvisionerFunc: func() tfplugin5.ProvisionerServer {
			return grpcwrap.Provisioner(localexec.New())
		},
	})
}

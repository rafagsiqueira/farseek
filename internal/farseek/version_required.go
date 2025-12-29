// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import (
	"github.com/rafagsiqueira/farseek/internal/tfdiags"

	"github.com/rafagsiqueira/farseek/internal/configs"
)

// CheckCoreVersionRequirements visits each of the modules in the given
// configuration tree and verifies that any given Core version constraints
// match with the version of Farseek Core that is being used.
//
// The returned diagnostics will contain errors if any constraints do not match.
// The returned diagnostics might also return warnings, which should be
// displayed to the user.
func CheckCoreVersionRequirements(config *configs.Config) tfdiags.Diagnostics {
	if config == nil {
		return nil
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(config.CheckCoreVersionRequirements())

	return diags
}

// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package eval

import (
	"github.com/rafagsiqueira/farseek/internal/addrs"
	"github.com/rafagsiqueira/farseek/internal/configs"
	"github.com/rafagsiqueira/farseek/internal/configs/configschema"
	"github.com/rafagsiqueira/farseek/internal/lang/eval/internal/evalglue"
	farseek2024 "github.com/rafagsiqueira/farseek/internal/lang/eval/internal/tofu2024"
	"github.com/rafagsiqueira/farseek/internal/providers"
)

// The symbols aliased in this file are defined in [evalglue] really just to
// avoid a dependency between this package and the "compiler" packages
// like ./internal/farseek2024, but we do still need them in our exported API
// here so that other parts of Farseek can interact with the evaluator.

type EvalContext = evalglue.EvalContext
type ProvidersSchema = evalglue.ProvidersSchema
type ProvisionersSchema = evalglue.ProvisionersSchema
type ExternalModules = evalglue.ExternalModules
type UncompiledModule = evalglue.UncompiledModule

func ModulesForTesting(modules map[addrs.ModuleSourceLocal]*configs.Module) ExternalModules {
	// This one actually lives in farseek2024 because evalglue isn't allowed to
	// depend on farseek2024 itself, but from the caller's perspective this is
	// still presented as an evalglue re-export because the return type belongs
	// to that package.
	return farseek2024.ModulesForTesting(modules)
}

func ProvidersForTesting(schemas map[addrs.Provider]*providers.GetProviderSchemaResponse) ProvidersSchema {
	return evalglue.ProvidersForTesting(schemas)
}

func ProvisionersForTesting(schemas map[string]*configschema.Block) ProvisionersSchema {
	return evalglue.ProvisionersForTesting(schemas)
}

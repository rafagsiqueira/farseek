// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package local

import (
	"context"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/rafagsiqueira/farseek/internal/backend"
	"github.com/rafagsiqueira/farseek/internal/plans"
	"github.com/zclconf/go-cty/cty"
)

func (b *Local) filterPlanChanges(
	ctx context.Context,
	op *backend.Operation,
	lr *backend.LocalRun,
	plan *plans.Plan,
) {
	if !op.FarseekMode || plan == nil || plan.Changes == nil {
		return
	}

	for _, rc := range plan.Changes.Resources {
		// Only care about updates
		if rc.Action != plans.Update {
			continue
		}

		// Find config
		addr := rc.Addr
		targetModuleConfig := lr.Config.DescendentForInstance(addr.Module)
		if targetModuleConfig == nil {
			continue
		}

		resConfig := targetModuleConfig.Module.ResourceByAddr(addr.Resource.Resource)
		if resConfig == nil {
			continue
		}

		// Get attributes present in config
		var inConfigFunc func(string) bool

		if syntaxBody, ok := resConfig.Config.(*hclsyntax.Body); ok {
			inConfigFunc = func(name string) bool {
				if _, ok := syntaxBody.Attributes[name]; ok {
					return true
				}
				for _, block := range syntaxBody.Blocks {
					if block.Type == name {
						return true
					}
				}
				return false
			}
		} else {
			// Fallback to JustAttributes
			configAttrs, _ := resConfig.Config.JustAttributes()
			inConfigFunc = func(name string) bool {
				_, ok := configAttrs[name]
				return ok
			}
		}

		// Reconstruct After value using schemas to handle blocks/nulls correctly
		schemas, schemaDiags := lr.Core.Schemas(ctx, lr.Config, lr.InputState)
		if schemaDiags.HasErrors() {
			continue
		}

		resSchemaBlock, _ := schemas.ResourceTypeConfig(resConfig.Provider, resConfig.Mode, resConfig.Type)
		if resSchemaBlock == nil {
			continue
		}

		ty := resSchemaBlock.ImpliedType()

		beforeVal, err := rc.Before.Decode(ty)
		if err != nil {
			continue
		}
		afterVal, err := rc.After.Decode(ty)
		if err != nil {
			continue
		}

		if !beforeVal.Type().IsObjectType() || !afterVal.Type().IsObjectType() {
			continue
		}

		beforeMap := beforeVal.AsValueMap()
		afterMap := afterVal.AsValueMap()
		newAfterMap := make(map[string]cty.Value)

		changed := false
		for attr, newVal := range afterMap {
			oldVal, existed := beforeMap[attr]

			// If values are equal, keep new val
			if existed && oldVal.RawEquals(newVal) {
				newAfterMap[attr] = newVal
				continue
			}

			// If changed (or new), check if in config
			if inConfigFunc(attr) {
				newAfterMap[attr] = newVal
			} else {
				// Not configured. If it existed before, revert to old value.
				if existed {
					newAfterMap[attr] = oldVal
					changed = true
				} else {
					// New attribute that wasn't there before (likely output-only)
					newAfterMap[attr] = newVal
				}
			}
		}

		if changed {
			newAfterVal := cty.ObjectVal(newAfterMap)
			newAfterEncoded, err := plans.NewDynamicValue(newAfterVal, ty)
			if err != nil {
				continue
			}
			rc.After = newAfterEncoded

			// Re-calculate Action
			if beforeVal.RawEquals(newAfterVal) {
				rc.Action = plans.NoOp
			}
		}
	}
}

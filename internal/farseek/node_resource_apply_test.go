// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import (
	"testing"

	"github.com/rafagsiqueira/farseek/internal/addrs"
	"github.com/rafagsiqueira/farseek/internal/configs"
	"github.com/rafagsiqueira/farseek/internal/instances"
	"github.com/rafagsiqueira/farseek/internal/states"
)

func TestNodeExpandApplyableResourceExecute(t *testing.T) {
	state := states.NewState()
	t.Run("no config", func(t *testing.T) {
		evalCtx := &MockEvalContext{
			StateState:               state.SyncWrapper(),
			InstanceExpanderExpander: instances.NewExpander(),
		}

		node := &nodeExpandApplyableResource{
			NodeAbstractResource: &NodeAbstractResource{
				Addr:   mustConfigResourceAddr("test_instance.foo"),
				Config: nil,
			},
		}
		diags := node.Execute(t.Context(), evalCtx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		state.PruneResourceHusks()
		if !state.Empty() {
			t.Fatalf("expected no state, got:\n %s", state.String())
		}
	})

	t.Run("simple", func(t *testing.T) {
		evalCtx := &MockEvalContext{
			StateState:               state.SyncWrapper(),
			InstanceExpanderExpander: instances.NewExpander(),
		}

		node := &nodeExpandApplyableResource{
			NodeAbstractResource: &NodeAbstractResource{
				Addr: mustConfigResourceAddr("test_instance.foo"),
				Config: &configs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_instance",
					Name: "foo",
				},
				ResolvedProvider: ResolvedProvider{ProviderConfig: addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				}},
			},
		}
		diags := node.Execute(t.Context(), evalCtx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}
		if state.Empty() {
			t.Fatal("expected resources in state, got empty state")
		}
		r := state.Resource(mustAbsResourceAddr("test_instance.foo"))
		if r == nil {
			t.Fatal("test_instance.foo not found in state")
		}
	})
}

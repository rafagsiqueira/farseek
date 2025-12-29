// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import (
	"context"

	"github.com/rafagsiqueira/farseek/internal/addrs"
	"github.com/rafagsiqueira/farseek/internal/configs"
	"github.com/rafagsiqueira/farseek/internal/dag"
)

// RootVariableTransformer is a GraphTransformer that adds all the root
// variables to the graph.
//
// Root variables are currently no-ops but they must be added to the
// graph since downstream things that depend on them must be able to
// reach them.
type RootVariableTransformer struct {
	Config *configs.Config

	RawValues InputValues
}

func (t *RootVariableTransformer) Transform(_ context.Context, g *Graph) error {
	// We can have no variables if we have no config.
	if t.Config == nil {
		return nil
	}

	// We're only considering root module variables here, since child
	// module variables are handled by ModuleVariableTransformer.
	vars := t.Config.Module.Variables

	// Add all variables here
	for _, v := range vars {
		node := &NodeRootVariable{
			Addr: addrs.InputVariable{
				Name: v.Name,
			},
			Config:   v,
			RawValue: t.RawValues[v.Name],
		}
		g.Add(node)

		ref := &nodeVariableReference{
			Addr: addrs.InputVariable{
				Name: v.Name,
			},
			Config: v,
		}
		g.Add(ref)

		// Input must be available before reference is valid
		g.Connect(dag.BasicEdge(ref, node))
	}

	return nil
}

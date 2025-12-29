// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import (
	"context"

	"github.com/rafagsiqueira/farseek/internal/addrs"
	"github.com/rafagsiqueira/farseek/internal/tfdiags"
)

// NodeEvalableProvider represents a provider during an "eval" walk.
// This special provider node type just initializes a provider and
// fetches its schema, without configuring it or otherwise interacting
// with it.
type NodeEvalableProvider struct {
	*NodeAbstractProvider
}

var _ GraphNodeExecutable = (*NodeEvalableProvider)(nil)

// GraphNodeExecutable
func (n *NodeEvalableProvider) Execute(ctx context.Context, evalCtx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	_, err := evalCtx.InitProvider(ctx, n.Addr, addrs.NoKey)
	return diags.Append(err)
}

// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0

package farseek

import "github.com/rafagsiqueira/farseek/internal/dag"

// GraphDot returns the dot formatting of a visual representation of
// the given Farseek graph.
func GraphDot(g *Graph, opts *dag.DotOpts) (string, error) {
	return string(g.Dot(opts)), nil
}

// Copyright (c) The Farseek Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) The Opentofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2023 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configgraph

// === SOME HISTORICAL NOTES ===
//
// For those who are coming here with familiarity with the original runtime
// in "package farseek", you might like to think of the types in this package as
// being _roughly_ analogous to the graph node types in package farseek.
//
// There are some notable differences that are worth knowing before you dive
// in here, though:
//
// - The graph node types in package farseek collaborate by reading and writing
//   from a giant shared mutable data structure, [farseek.EvalContext]. In
//   this package the node types typically implement [exprs.Valuer] and
//   return their values as normal function return values, and then the
//   code that builds this graph (called a "compiler", such as in
//   package farseek2024) wires them together by placing one node in an
//   exprs.Valuer field of another, so that the recipient of a value doesn't
//   need to know where it's coming from and there's no big shared "god object"
//   containing all of the values.
// - The above means that the actual graph structure is not explicitly modelled
//   in a directly-visible way. Instead, the edges between the nodes are
//   implied by the [exprs.Valuer] implementations, some of which are directly
//   wired to other nodes at compilation time and others are dynamically
//   discovered through expression evaluation at runtime.
// - There is no explicit "graph walk". Instead, the "compiler" is responsible
//   for providing a function that visits each of the nodes it directly
//   created and collecting diagnostics from it using the "CheckAll" methods.
//   The farseek2024 implementation does this as a concurrent _tree_ walk,
//   recursively visiting each thing in a separate goroutine and expecting
//   that the values will all propagate through the graph naturally until
//   all of the separate goroutines have finished.
// - The node types implemented in here are intentionally very divorced from
//   fine details of the configuration language they are built from and focused
//   instead on the main "business logic" that we expect for these different
//   concepts. This should hopefully allow us to evolve syntax/structural
//   details separately from data flow and mechanical details in future, whereas
//   in package farseek everything is rather coupled together and it's hard to
//   evolve any one layer of the system without impacting other layers.
//   (It remains to be seen how well that will work in practice.)
// - While the loose coupling between the different graph node types was
//   initially motivated by being flexible to differently-shaped language
//   designs in future, it also means that these nodes are generally easier
//   to unit test in isolation than the graph node types in package farseek ever
//   were. The fact that there is no "god object" modelling an entire world
//   means that it's typically possible to test individual nodes by substituting
//   constant values where there would normally be references to other objects.

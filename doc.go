// Package hyperpaths implements the Spiess-Florian (1989) optimal-strategies
// transit assignment ("hyperpaths").
//
// # Two API tiers
//
// The package offers the same algorithm through two interfaces. They produce
// identical results; pick by use case.
//
// 1. Simple string API - [ComputeSF], [FindOptimalStrategy], [AssignDemand].
// Nodes are plain string names, the OD is map[origin]map[dest]float64, and
// results come back as human-readable maps ([Strategy.Labels],
// [Strategy.Freqs], [Volumes.Links]). One call, nothing to set up. This is the
// reference / debugging path: it is the easiest to read, and setting the
// package variable [Verbose] to true prints a step-by-step trace of both
// phases. Internally it already uses the same integer arena as the fast path,
// so a single solve is fast. Use it for one-off or single-destination solves,
// small networks, debugging, and first integration.
//
// 2. Arena API - [Graph], [Workspace], [Workspace.Assign], [Workspace.SolveEach].
// [NewGraph] interns the network into an integer arena once; a [Workspace]
// holds reusable buffers so each destination is assigned with no further
// allocation, and results are returned in integer (arena) indexing. Use it for
// assigning many destinations, large networks, and long-running or multi-user
// services, where rebuilding the graph and allocating result maps per
// destination dominate. On a full assignment (every stop a destination) it is
// an order of magnitude faster than calling [ComputeSF] per destination, and
// allocation-free once the Workspace is warm.
//
// # Which one
//
//   - Debugging, reference, or a single destination: [ComputeSF] (with [Verbose]).
//   - Production, many destinations, servers: [NewGraph] + [Graph.NewWorkspace] + [Workspace.SolveEach].
//
// The string API keeps working unchanged; adopting the arena API is opt-in.
//
// # Concurrency
//
// A [Graph] is immutable after [NewGraph] and safe to share across goroutines.
// A [Workspace] is mutable and must not be shared; give each goroutine its own,
// or pool them:
//
//	graph := hyperpaths.NewGraph(links, stops)
//	pool := sync.Pool{New: func() any { return graph.NewWorkspace() }}
//	// per request (own goroutine):
//	w := pool.Get().(*hyperpaths.Workspace)
//	defer pool.Put(w)
//	w.SolveEach(od, func(res *hyperpaths.DestResult) { /* use res */ })
//
// The [DestResult] and the slices returned by [Workspace.Assign] /
// [Workspace.SolveEach] are owned by the Workspace and reused on the next call,
// so copy out anything that must outlive it. Do not mutate a shared network,
// and do not toggle [Verbose], while assignments are running.
package hyperpaths

package hyperpaths

import "math"

// Graph is an immutable, interned transit network (integer arena). Build it
// once with NewGraph and share it across goroutines: it is read-only, so
// concurrent assignments are safe as long as each uses its own Workspace.
//
// Splitting the immutable graph from the mutable per-solve buffers is what
// makes the multi-destination / multi-request server case cheap: the string
// interning and adjacency are built a single time, and each destination is
// then assigned by reusing a Workspace with no further allocation.
//
// In a concurrent server, build one Graph at startup and pool the Workspaces,
// one per in-flight request:
//
//	graph := hyperpaths.NewGraph(links, stops) // once, shared, read-only
//	pool := sync.Pool{New: func() any { return graph.NewWorkspace() }}
//
//	// per request (own goroutine):
//	w := pool.Get().(*hyperpaths.Workspace)
//	defer pool.Put(w)
//	w.SolveEach(od, func(res *hyperpaths.DestResult) {
//		// accumulate res.LinkVol before the next destination
//	})
//
// The Graph is shared safely because it is never mutated; each request mutates
// only its own Workspace.
type Graph struct {
	// arena index -> node name
	nName []string
	// node name -> arena index
	nID map[string]int
	// number of nodes
	n int
	// number of links
	m int
	// link -> tail node index
	from []int
	// link -> head node index
	to []int
	// link -> travel cost
	cost []float64
	// link -> headway (0 = no-wait / infinite frequency)
	head []float64
	// node -> indices of links whose head is that node
	adjByTo [][]int
}

// NewGraph interns the network once. allStops is interned first, so node
// indices [0, len(allStops)) are the stop nodes; any link endpoint outside
// allStops (out of contract) is appended after.
func NewGraph(allLinks []*Link, allStops map[string]struct{}) *Graph {
	g := &Graph{nID: make(map[string]int, len(allStops))}
	intern := func(name string) int {
		if id, ok := g.nID[name]; ok {
			return id
		}
		id := len(g.nName)
		g.nID[name] = id
		g.nName = append(g.nName, name)
		return id
	}
	for stop := range allStops {
		intern(stop)
	}
	g.m = len(allLinks)
	g.from = make([]int, g.m)
	g.to = make([]int, g.m)
	g.cost = make([]float64, g.m)
	g.head = make([]float64, g.m)
	for k, link := range allLinks {
		g.from[k] = intern(link.FromNode)
		g.to[k] = intern(link.ToNode)
		g.cost[k] = link.TravelCost
		g.head[k] = link.Headway
	}
	g.n = len(g.nName)
	g.adjByTo = make([][]int, g.n)
	for k := 0; k < g.m; k++ {
		g.adjByTo[g.to[k]] = append(g.adjByTo[g.to[k]], k)
	}
	return g
}

// NumNodes and NumLinks expose the arena sizes (to size demand buffers).
func (g *Graph) NumNodes() int { return g.n }
func (g *Graph) NumLinks() int { return g.m }

// NodeIndex returns the arena index of a node name, or -1 if unknown.
func (g *Graph) NodeIndex(name string) int {
	if id, ok := g.nID[name]; ok {
		return id
	}
	return -1
}

// NodeName returns the name of an arena node index.
func (g *Graph) NodeName(id int) string { return g.nName[id] }

// DestResult is one destination's assignment in arena (integer) indexing.
// All slices are owned by the Workspace and reused on the next Assign call,
// so copy out anything that must outlive it.
type DestResult struct {
	DestID  int
	Labels  []float64 // node -> expected time to destination (u_i)
	Freqs   []float64 // node -> combined attractive frequency (f_i); +Inf = no-wait
	ASet    []int     // accepted link indices, in acceptance order
	LinkVol []float64 // link -> assigned volume
	NodeVol []float64 // node -> accumulated volume
}

// Workspace holds the reusable per-solve buffers for one Graph. Create one
// per goroutine (or pool them); it is NOT safe for concurrent use.
type Workspace struct {
	g *Graph

	u    []float64
	f    []float64
	slab []pqEntry
	pq   PriorityQueue

	overlineA []int
	aSetIdx   [][]int
	aSet      []int

	linkVol []float64
	nodeVol []float64

	// reused per-destination demand column for SolveEach
	demand []float64
	// reused result view returned by Assign
	result DestResult
	// reused per-destination OD columns for SolveEach
	cols [][]odEntry
}

// odEntry is one origin's demand toward a destination, used by SolveEach's
// reusable transpose of the OD matrix.
type odEntry struct {
	origin int
	demand float64
}

// NewWorkspace allocates the working buffers for g once; reuse it across
// destinations and requests.
func (g *Graph) NewWorkspace() *Workspace {
	return &Workspace{
		g:         g,
		u:         make([]float64, g.n),
		f:         make([]float64, g.n),
		slab:      make([]pqEntry, g.m),
		pq:        make(PriorityQueue, 0, g.m),
		overlineA: make([]int, 0, g.m/2),
		aSetIdx:   make([][]int, g.n),
		aSet:      make([]int, 0, g.m/2),
		linkVol:   make([]float64, g.m),
		nodeVol:   make([]float64, g.n),
		demand:    make([]float64, g.n),
		cols:      make([][]odEntry, g.n),
	}
}

// findStrategy runs Spiess-Florian phase 1 for destID into u, f, overlineA
// and aSetIdx, reusing the workspace buffers. Identical algorithm to
// FindOptimalStrategy, just arena-indexed and allocation-free.
func (w *Workspace) findStrategy(destID int) {
	g := w.g
	for id := 0; id < g.n; id++ {
		w.f[id] = 0.0
		if id == destID {
			w.u[id] = 0.0
		} else {
			w.u[id] = math.Inf(+1)
		}
		w.aSetIdx[id] = w.aSetIdx[id][:0]
	}
	w.overlineA = w.overlineA[:0]

	w.pq = w.pq[:0]
	for k := 0; k < g.m; k++ {
		w.slab[k] = pqEntry{link: k, priority: w.u[g.to[k]] + g.cost[k]}
		w.pq = append(w.pq, &w.slab[k])
	}
	w.pq.Init()

	for w.pq.Len() > 0 {
		entry := w.pq.Pop().(*pqEntry)
		if math.IsInf(entry.priority, 1) {
			break
		}
		k := entry.link
		i := g.from[k]
		j := g.to[k]
		sumUC := w.u[j] + g.cost[k]

		if math.IsInf(w.f[i], 1) {
			continue
		}
		if w.u[i] <= sumUC {
			continue
		}
		if g.head[k] <= 0 {
			w.u[i] = sumUC
			w.f[i] = math.Inf(+1)
			for _, idx := range w.aSetIdx[i] {
				w.overlineA[idx] = -1
			}
			w.aSetIdx[i] = w.aSetIdx[i][:0]
			w.overlineA = append(w.overlineA, k)
			w.aSetIdx[i] = append(w.aSetIdx[i], len(w.overlineA)-1)
		} else {
			freq := 1.0 / g.head[k]
			if w.f[i] == 0 {
				w.u[i] = (alpha + freq*sumUC) / freq
			} else {
				w.u[i] = (w.f[i]*w.u[i] + freq*sumUC) / (w.f[i] + freq)
			}
			w.f[i] += freq
			w.overlineA = append(w.overlineA, k)
			w.aSetIdx[i] = append(w.aSetIdx[i], len(w.overlineA)-1)
		}

		for _, kk := range g.adjByTo[i] {
			w.pq.update(&w.slab[kk], w.u[i]+g.cost[kk])
		}
	}

	// Compact the attractive set (drop -1 slots), preserving acceptance order.
	w.aSet = w.aSet[:0]
	for _, k := range w.overlineA {
		if k >= 0 {
			w.aSet = append(w.aSet, k)
		}
	}
}

// Assign runs the full Spiess-Florian assignment (optimal strategy + demand
// loading) for one destination index. demand is a per-node slice of trips
// heading to destID (demand[destID] is ignored). It returns arena-indexed
// results backed by the workspace buffers; they are valid until the next
// Assign call on this workspace.
func (w *Workspace) Assign(destID int, demand []float64) *DestResult {
	g := w.g
	w.findStrategy(destID)

	// Phase 2: seed node volumes from the demand column; the destination
	// absorbs flow (negated so arrivals cancel it to zero).
	total := 0.0
	for id := 0; id < g.n; id++ {
		if id != destID && demand[id] != 0 {
			w.nodeVol[id] = demand[id]
			total += demand[id]
		} else {
			w.nodeVol[id] = 0
		}
	}
	w.nodeVol[destID] = -total

	for k := 0; k < g.m; k++ {
		w.linkVol[k] = 0
	}

	// Load in reverse acceptance order (decreasing u_j + c_a, p. 97).
	for idx := len(w.aSet) - 1; idx >= 0; idx-- {
		k := w.aSet[idx]
		i := g.from[k]
		fi := w.f[i]
		var va float64
		if math.IsInf(fi, 1) {
			va = w.nodeVol[i]
		} else {
			freq := 1.0 / g.head[k]
			va = (freq / fi) * w.nodeVol[i]
		}
		w.linkVol[k] = va
		w.nodeVol[g.to[k]] += va
	}

	w.result = DestResult{
		DestID:  destID,
		Labels:  w.u,
		Freqs:   w.f,
		ASet:    w.aSet,
		LinkVol: w.linkVol,
		NodeVol: w.nodeVol,
	}
	return &w.result
}

// SolveEach assigns every destination present in od (an
// origin -> destination -> demand matrix) and calls fn with the arena-indexed
// result for each. It transposes od into per-destination demand columns once,
// resolving node names to indices a single time, then reuses the workspace
// buffers, so there are no per-destination allocations. The DestResult given
// to fn is reused between calls, so copy out anything that must outlive the
// callback.
func (w *Workspace) SolveEach(od map[string]map[string]float64, fn func(res *DestResult)) {
	g := w.g
	// Transpose od into per-destination columns, reusing the workspace's
	// backing slices (only their lengths are reset, so after warm-up this
	// allocates nothing).
	for i := range w.cols {
		w.cols[i] = w.cols[i][:0]
	}
	for origin, row := range od {
		oid := g.NodeIndex(origin)
		if oid < 0 {
			continue
		}
		for dest, d := range row {
			if d == 0 {
				continue
			}
			did := g.NodeIndex(dest)
			if did < 0 {
				continue
			}
			w.cols[did] = append(w.cols[did], odEntry{oid, d})
		}
	}
	for did := 0; did < g.n; did++ {
		entries := w.cols[did]
		if len(entries) == 0 {
			continue
		}
		for i := range w.demand {
			w.demand[i] = 0
		}
		for _, e := range entries {
			w.demand[e.origin] = e.demand
		}
		fn(w.Assign(did, w.demand))
	}
}

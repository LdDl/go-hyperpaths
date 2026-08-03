package hyperpaths

import (
	"fmt"
	"testing"
)

// genGridNetwork builds a synthetic expanded transit route graph on a
// rows x cols grid of stops. Each row has an eastbound line and each column a
// southbound line; every line is expanded the Spiess-Florian way into
// boarding links (stop -> platform, headway > 0), riding links
// (platform -> platform, in-vehicle time, no wait) and alighting links
// (platform -> stop, no wait). The bottom-right stop is the destination and
// is reachable from every stop, and the shared grid stops create genuine
// common-line choices - so the hyperpath is non-trivial.
//
// It returns the link list, the node set, the destination stop and an OD
// matrix loading one trip from every other stop to the destination.
func genGridNetwork(rows, cols int, headway, segTime float64) ([]*Link, map[string]struct{}, string, map[string]map[string]float64) {
	stop := func(r, c int) string { return fmt.Sprintf("s_%d_%d", r, c) }

	nodes := make(map[string]struct{})
	var links []*Link

	// Register every stop node.
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			nodes[stop(r, c)] = struct{}{}
		}
	}

	// Expand one line over an ordered list of stops into board/ride/alight.
	addLine := func(id string, seq []string) {
		platform := func(i int) string { return fmt.Sprintf("%s#%d", id, i) }
		for i := range seq {
			nodes[platform(i)] = struct{}{}
			if i < len(seq)-1 {
				// Boarding: stop -> platform (wait = headway).
				links = append(links, &Link{seq[i], platform(i), id, 0, headway})
				// Riding: platform -> next platform (in-vehicle, no wait).
				links = append(links, &Link{platform(i), platform(i + 1), id, segTime, 0})
			}
			if i > 0 {
				// Alighting: platform -> stop (no wait, no cost).
				links = append(links, &Link{platform(i), seq[i], id, 0, 0})
			}
		}
	}

	for r := 0; r < rows; r++ {
		seq := make([]string, cols)
		for c := 0; c < cols; c++ {
			seq[c] = stop(r, c)
		}
		addLine(fmt.Sprintf("R%d", r), seq)
	}
	for c := 0; c < cols; c++ {
		seq := make([]string, rows)
		for r := 0; r < rows; r++ {
			seq[r] = stop(r, c)
		}
		addLine(fmt.Sprintf("C%d", c), seq)
	}

	dest := stop(rows-1, cols-1)
	od := map[string]map[string]float64{}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			s := stop(r, c)
			if s == dest {
				continue
			}
			od[s] = map[string]float64{dest: 1.0}
		}
	}
	return links, nodes, dest, od
}

var benchSizes = []struct {
	name       string
	rows, cols int
}{
	{"8x8", 8, 8},
	{"16x16", 16, 16},
	{"32x32", 32, 32},
}

func BenchmarkFindOptimalStrategy(b *testing.B) {
	for _, sz := range benchSizes {
		links, nodes, dest, _ := genGridNetwork(sz.rows, sz.cols, 6.0, 3.0)
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = FindOptimalStrategy(links, nodes, dest)
			}
		})
	}
}

func BenchmarkComputeSF(b *testing.B) {
	for _, sz := range benchSizes {
		links, nodes, dest, od := genGridNetwork(sz.rows, sz.cols, 6.0, 3.0)
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = ComputeSF(links, nodes, dest, od)
			}
		})
	}
}

func gridStops(rows, cols int) []string {
	stops := make([]string, 0, rows*cols)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			stops = append(stops, fmt.Sprintf("s_%d_%d", r, c))
		}
	}
	return stops
}

var multiSizes = []struct {
	name       string
	rows, cols int
}{
	{"8x8", 8, 8},
	{"12x12", 12, 12},
}

// Full assignment to every stop as a destination, the current way: ComputeSF
// per destination, which re-interns the network and rebuilds the graph every
// time.
func BenchmarkMultiDestComputeSF(b *testing.B) {
	for _, sz := range multiSizes {
		links, nodes, _, _ := genGridNetwork(sz.rows, sz.cols, 6.0, 3.0)
		stops := gridStops(sz.rows, sz.cols)
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				for _, dest := range stops {
					od := make(map[string]map[string]float64, len(stops))
					for _, o := range stops {
						if o != dest {
							od[o] = map[string]float64{dest: 1.0}
						}
					}
					_ = ComputeSF(links, nodes, dest, od)
				}
			}
		})
	}
}

// Same assignment via the arena Graph/Workspace: the Graph is interned once
// per full assignment and one Workspace is reused across destinations.
func BenchmarkMultiDestSolver(b *testing.B) {
	for _, sz := range multiSizes {
		links, nodes, _, _ := genGridNetwork(sz.rows, sz.cols, 6.0, 3.0)
		stops := gridStops(sz.rows, sz.cols)
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				g := NewGraph(links, nodes)
				w := g.NewWorkspace()
				demand := make([]float64, g.NumNodes())
				stopIDs := make([]int, len(stops))
				for k, s := range stops {
					stopIDs[k] = g.NodeIndex(s)
				}
				for _, destID := range stopIDs {
					for k := range demand {
						demand[k] = 0
					}
					for _, oid := range stopIDs {
						if oid != destID {
							demand[oid] = 1.0
						}
					}
					_ = w.Assign(destID, demand)
				}
			}
		})
	}
}

// The server case: a static network, so the Graph is interned once for all
// requests; only the Workspace and the per-destination Assign are per request.
func BenchmarkMultiDestSolverCachedGraph(b *testing.B) {
	for _, sz := range multiSizes {
		links, nodes, _, _ := genGridNetwork(sz.rows, sz.cols, 6.0, 3.0)
		stops := gridStops(sz.rows, sz.cols)
		g := NewGraph(links, nodes)
		stopIDs := make([]int, len(stops))
		for k, s := range stops {
			stopIDs[k] = g.NodeIndex(s)
		}
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				w := g.NewWorkspace()
				demand := make([]float64, g.NumNodes())
				for _, destID := range stopIDs {
					for k := range demand {
						demand[k] = 0
					}
					for _, oid := range stopIDs {
						if oid != destID {
							demand[oid] = 1.0
						}
					}
					_ = w.Assign(destID, demand)
				}
			}
		})
	}
}

func gridFullOD(stops []string) map[string]map[string]float64 {
	od := make(map[string]map[string]float64, len(stops))
	for _, o := range stops {
		row := make(map[string]float64, len(stops)-1)
		for _, d := range stops {
			if d != o {
				row[d] = 1.0
			}
		}
		od[o] = row
	}
	return od
}

// The ergonomic server case: static Graph cached, a full OD matrix handed to
// SolveEach, which transposes it once and reuses the Workspace. Should match
// the hand-written MultiDestSolverCachedGraph loop.
func BenchmarkMultiDestSolveEach(b *testing.B) {
	for _, sz := range multiSizes {
		links, nodes, _, _ := genGridNetwork(sz.rows, sz.cols, 6.0, 3.0)
		stops := gridStops(sz.rows, sz.cols)
		od := gridFullOD(stops)
		g := NewGraph(links, nodes)
		// Static network + pooled/reused Workspace: the steady-state server
		// case, where the transpose buffers are already warm.
		w := g.NewWorkspace()
		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				w.SolveEach(od, func(res *DestResult) { _ = res })
			}
		})
	}
}

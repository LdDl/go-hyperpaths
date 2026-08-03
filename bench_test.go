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

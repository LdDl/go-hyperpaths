package hyperpaths

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

// TestSolverParity checks that the arena Graph/Workspace assignment produces
// exactly the same labels and link volumes as the string-keyed ComputeSF, on
// the 4x4 grid to the corner destination.
func TestSolverParity(t *testing.T) {
	links, nodes, dest, od := genGridNetwork(4, 4, 6.0, 3.0)

	ref := ComputeSF(links, nodes, dest, od)

	g := NewGraph(links, nodes)
	w := g.NewWorkspace()
	destID := g.NodeIndex(dest)

	// demand column: 1 trip from every origin that has a trip to dest in od.
	demand := make([]float64, g.NumNodes())
	for origin := range od {
		if v, ok := od[origin][dest]; ok {
			demand[g.NodeIndex(origin)] = v
		}
	}
	got := w.Assign(destID, demand)

	const eps = 1e-9

	// Labels and freqs match for every node.
	for name, want := range ref.Strategy.Labels {
		id := g.NodeIndex(name)
		if id < 0 {
			t.Fatalf("node %s missing in graph", name)
		}
		if math.Abs(got.Labels[id]-want) > eps {
			t.Errorf("label[%s] = %.15g, want %.15g", name, got.Labels[id], want)
		}
		if math.Abs(got.Freqs[id]-ref.Strategy.Freqs[name]) > eps &&
			!(math.IsInf(got.Freqs[id], 1) && math.IsInf(ref.Strategy.Freqs[name], 1)) {
			t.Errorf("freq[%s] = %.15g, want %.15g", name, got.Freqs[id], ref.Strategy.Freqs[name])
		}
	}

	// Link volumes match for every link (compare per (from,to)).
	for k, link := range links {
		want := ref.Volumes.Links[link.FromNode][link.ToNode]
		if math.Abs(got.LinkVol[k]-want) > eps {
			t.Errorf("linkVol[%s->%s] = %.15g, want %.15g",
				link.FromNode, link.ToNode, got.LinkVol[k], want)
		}
	}
}

// TestSolveEachParity checks that SolveEach over a full OD matrix accumulates
// exactly the same total link volume as calling ComputeSF once per
// destination, on the 4x4 grid.
func TestSolveEachParity(t *testing.T) {
	rows, cols := 4, 4
	links, nodes, _, _ := genGridNetwork(rows, cols, 6.0, 3.0)

	stops := gridStops(rows, cols)
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

	// Reference total via the string API, one ComputeSF per destination.
	wantTotal := 0.0
	for _, dest := range stops {
		col := make(map[string]map[string]float64, len(stops))
		for _, o := range stops {
			if o != dest {
				col[o] = map[string]float64{dest: 1.0}
			}
		}
		res := ComputeSF(links, nodes, dest, col)
		for _, m := range res.Volumes.Links {
			for _, v := range m {
				wantTotal += v
			}
		}
	}

	// SolveEach total.
	g := NewGraph(links, nodes)
	w := g.NewWorkspace()
	gotTotal := 0.0
	w.SolveEach(od, func(res *DestResult) {
		for _, v := range res.LinkVol {
			gotTotal += v
		}
	})

	if math.Abs(gotTotal-wantTotal) > 1e-6 {
		t.Errorf("SolveEach total volume = %.15g, want %.15g", gotTotal, wantTotal)
	}
}

// TestConcurrentPool runs many goroutines that share one immutable Graph and
// each take a Workspace from a sync.Pool, proving the server pattern is both
// correct and race-free (run with -race). Every goroutine must reproduce the
// single-threaded reference total.
func TestConcurrentPool(t *testing.T) {
	links, nodes, _, _ := genGridNetwork(6, 6, 6.0, 3.0)
	stops := gridStops(6, 6)
	od := gridFullOD(stops)
	g := NewGraph(links, nodes)

	// Single-threaded reference.
	wRef := g.NewWorkspace()
	var refTotal float64
	wRef.SolveEach(od, func(res *DestResult) {
		for _, v := range res.LinkVol {
			refTotal += v
		}
	})

	pool := sync.Pool{New: func() any { return g.NewWorkspace() }}
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := pool.Get().(*Workspace)
			defer pool.Put(w)
			var total float64
			w.SolveEach(od, func(res *DestResult) {
				for _, v := range res.LinkVol {
					total += v
				}
			})
			if math.Abs(total-refTotal) > 1e-6 {
				errs <- fmt.Errorf("goroutine total %.15g != ref %.15g", total, refTotal)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

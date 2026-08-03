package hyperpaths

import (
	"math"
	"testing"
)

// TestGoldenGrid locks the exact result of ComputeSF on a fixed 4x4 synthetic
// grid network. The anchor values were captured from the reference
// implementation; any change to the solver (e.g. the integer-arena rewrite)
// must reproduce them exactly, so this guards the optimization against
// altering results.
func TestGoldenGrid(t *testing.T) {
	links, nodes, dest, od := genGridNetwork(4, 4, 6.0, 3.0)
	res := ComputeSF(links, nodes, dest, od)

	const eps = 1e-9
	check := func(name string, got, want float64) {
		if math.Abs(got-want) > eps {
			t.Errorf("%s = %.15g, want %.15g", name, got, want)
		}
	}

	check("label[s_0_0]", res.Strategy.Labels["s_0_0"], 27.0)
	check("label[s_2_1]", res.Strategy.Labels["s_2_1"], 18.0)
	check("freq[s_0_0]", res.Strategy.Freqs["s_0_0"], 1.0/3.0)

	if len(res.Strategy.ASet) != 56 {
		t.Errorf("aSet len = %d, want 56", len(res.Strategy.ASet))
	}

	total := 0.0
	for _, m := range res.Volumes.Links {
		for _, v := range m {
			total += v
		}
	}
	check("total volume", total, 96.0)
}

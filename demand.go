package hyperpaths

import (
	"fmt"
	"math"
)

// Volumes holds the assigned demand according to the optimal strategy.
type Volumes struct {
	// Link volumes: Links[fromNode][toNode] = flow
	Links map[string]map[string]float64
	// Node volumes: accumulated flow through each node
	Nodes map[string]float64
}

func AssignDemand(allLinks []*Link, allStops map[string]struct{}, optimalStrategy *Strategy, trips map[string]map[string]float64, destination string) *Volumes {
	// The attractive set is built in acceptance order, which is
	// non-decreasing u_j + c_a (heap pops), so its reverse is exactly the
	// paper's decreasing loading order - no sorting needed, as the paper
	// notes on p. 97: the processing order of step 2.2 "is the inverse of
	// the order used in part 1 of the algorithm". At zero-cost ties
	// (no-wait chains produce exactly equal keys) reverse acceptance
	// order also guarantees that a node's inflow links are loaded before
	// its outflow links: a link (i, j) is accepted before the links into i
	// are updated and popped.
	sorted := optimalStrategy.ASet

	nodeVolumes := make(map[string]float64, len(allStops))
	for i := range allStops {
		nodeVolumes[i] = 0
	}
	for origin := range trips {
		if tripsNum, ok := trips[origin][destination]; ok {
			nodeVolumes[origin] = tripsNum
			nodeVolumes[destination] += tripsNum
		}
	}
	// Destination absorbs flow: negate so arrivals cancel it to zero.
	nodeVolumes[destination] *= -1

	v := make(map[string]map[string]float64)
	for _, a := range allLinks {
		if _, ok := v[a.FromNode]; !ok {
			v[a.FromNode] = make(map[string]float64)
		}
		v[a.FromNode][a.ToNode] = 0.0
	}

	for k := len(sorted) - 1; k >= 0; k-- {
		a := sorted[k]
		fi := optimalStrategy.Freqs[a.FromNode]
		var va float64
		if math.IsInf(fi, 1) {
			// A no-wait basket holds exactly one link (the one that replaced
			// it); per the paper's modified step 2.2 (p. 96) the link takes
			// the whole node volume: v_a := V_i
			va = nodeVolumes[a.FromNode]
		} else {
			// A finite basket holds only boarding links (headway > 0)
			freq := 1.0 / a.Headway
			va = (freq / fi) * nodeVolumes[a.FromNode]
		}
		if Verbose {
			fmt.Printf("Assigning demand for link: (%s, %s) \\\\ \n", a.FromNode, a.ToNode)
			fmt.Printf("\\quad $v_{(%s, %s)} = %v$, $V_{%s} = %v + %v$ \\\\ \n", a.FromNode, a.ToNode, va, a.ToNode, nodeVolumes[a.ToNode], va)
		}
		v[a.FromNode][a.ToNode] = va
		nodeVolumes[a.ToNode] += va
	}
	if Verbose {
		fmt.Println("Final node volumes: \\\\")
		for k := range nodeVolumes {
			fmt.Printf("\\quad $V_{%s} = %v$ \\\\ \n", k, nodeVolumes[k])
		}
	}

	result := &Volumes{
		Links: v,
		Nodes: nodeVolumes,
	}
	return result
}

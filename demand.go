package hyperpaths

import (
	"fmt"
	"sort"
)

// Volumes holds the assigned demand according to the optimal strategy.
type Volumes struct {
	// Link volumes: Links[fromNode][toNode] = flow
	Links map[string]map[string]float64
	// Node volumes: accumulated flow through each node
	Nodes map[string]float64
}

func AssignDemand(allLinks []*Link, allStops map[string]struct{}, optimalStrategy *Strategy, trips map[string]map[string]float64, destination string) *Volumes {
	// Work on a copy so the caller's ASet order is preserved.
	sorted := make([]*Link, len(optimalStrategy.ASet))
	copy(sorted, optimalStrategy.ASet)

	// Sort attractive links by decreasing (u_j + c_a).
	// Tie-break by decreasing u[FromNode] so upstream nodes are loaded first.
	sort.SliceStable(sorted, func(i, j int) bool {
		ai := optimalStrategy.Labels[sorted[i].ToNode] + sorted[i].TravelCost
		aj := optimalStrategy.Labels[sorted[j].ToNode] + sorted[j].TravelCost
		if ai != aj {
			return ai > aj
		}
		return optimalStrategy.Labels[sorted[i].FromNode] > optimalStrategy.Labels[sorted[j].FromNode]
	})

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

	for _, a := range sorted {
		freq := infiniteFrequency
		if a.Headway > 0 {
			freq = 1.0 / a.Headway
		}
		va := (freq / optimalStrategy.Freqs[a.FromNode]) * nodeVolumes[a.FromNode]
		if Verbose {
			fmt.Printf("Assigning demand for link: (%s, %s) \\\\ \n", a.FromNode, a.ToNode)
			fmt.Printf("\\quad $v_{(%s, %s)} = \\frac{%v}{%v}%v = %v$ \\\\ \n", a.FromNode, a.ToNode, freq, optimalStrategy.Freqs[a.FromNode], nodeVolumes[a.FromNode], va)
			fmt.Printf("\\quad $V_{%s} = V_{%s} + v_{(%s, %s) = %v + %v = %v}$ \\\\ \n", a.ToNode, a.ToNode, a.FromNode, a.ToNode, nodeVolumes[a.ToNode], va, nodeVolumes[a.ToNode]+va)
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

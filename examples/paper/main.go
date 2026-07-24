package main

import (
	"fmt"

	"github.com/lddl/go-hyperpaths"
)

func main() {
	allNodes := map[string]struct{}{
		"A": {},
		"X": {}, "X2": {},
		"Y": {}, "Y3": {},
		"B": {},
	}
	allLinks := []*hyperpaths.Link{
		{"A", "B", "Line 1", 25, 6},
		{"A", "X2", "Line 2", 7, 6},
		{"X2", "X", "Line 2", 0, 0},
		{"X", "X2", "Line 2", 0, 6},
		{"X2", "Y", "Line 2", 6, 0},
		{"Y3", "Y", "Line 3", 0, 15},
		{"Y", "B", "Line 4", 10, 3},
		{"X", "Y3", "Line 3", 4, 15},
		{"Y", "Y3", "Line 3", 0, 15},
		{"Y3", "B", "Line 3", 4, 0},
	}
	destinationNode := "B"
	odMatrix := map[string]map[string]float64{
		"A": {
			"B": 1,
		},
	}
	res := hyperpaths.ComputeSF(allLinks, allNodes, destinationNode, odMatrix)
	fmt.Println("Optimal strategy:")
	fmt.Println("\tNode labels:")
	for nodeID, nodeLabel := range res.Strategy.Labels {
		fmt.Printf("\t\tu_{i} = %s: %f\n", nodeID, nodeLabel)
	}
	fmt.Println("\tNodes probablities:")
	for nodeID, freq := range res.Strategy.Freqs {
		fmt.Printf("\t\tf_{i} = %s: %f\n", nodeID, freq)
	}
	fmt.Println("\tAttractive links set:")
	for _, link := range res.Strategy.ASet {
		fmt.Printf("\t\t a = (i, j) = (%s, %s)\n", link.FromNode, link.ToNode)
	}
	fmt.Println("Volumes:")
	fmt.Println("\tLinks volumes:")
	for fromNode := range res.Volumes.Links {
		for toNode, volume := range res.Volumes.Links[fromNode] {
			fmt.Printf("\t\tv_{i, j} = (%s, %s): %f\n", fromNode, toNode, volume)
		}
	}
	fmt.Println("\tNodes volumes:")
	for nodeID, volume := range res.Volumes.Nodes {
		fmt.Printf("\t\tv_{i} = %s: %f\n", nodeID, volume)
	}
}

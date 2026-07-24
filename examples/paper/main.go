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
		{FromNode: "A", ToNode: "B", RouteID: "Line 1", TravelCost: 25, Headway: 6},
		{FromNode: "A", ToNode: "X2", RouteID: "Line 2", TravelCost: 7, Headway: 6},
		{FromNode: "X2", ToNode: "X", RouteID: "Line 2", TravelCost: 0, Headway: 0},
		{FromNode: "X", ToNode: "X2", RouteID: "Line 2", TravelCost: 0, Headway: 6},
		{FromNode: "X2", ToNode: "Y", RouteID: "Line 2", TravelCost: 6, Headway: 0},
		{FromNode: "Y3", ToNode: "Y", RouteID: "Line 3", TravelCost: 0, Headway: 15},
		{FromNode: "Y", ToNode: "B", RouteID: "Line 4", TravelCost: 10, Headway: 3},
		{FromNode: "X", ToNode: "Y3", RouteID: "Line 3", TravelCost: 4, Headway: 15},
		{FromNode: "Y", ToNode: "Y3", RouteID: "Line 3", TravelCost: 0, Headway: 15},
		{FromNode: "Y3", ToNode: "B", RouteID: "Line 3", TravelCost: 4, Headway: 0},
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

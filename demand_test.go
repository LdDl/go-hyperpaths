package hyperpaths

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssignDemand(t *testing.T) {
	allNodes := map[string]struct{}{
		"A": {},
		"X": {}, "X2": {},
		"Y": {}, "Y3": {},
		"B": {},
	}
	allLinks := []*Link{
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
	optimalStrategy := Strategy{
		Labels: map[string]float64{
			"A":  27.75,
			"X":  19.071428571428573,
			"X2": 17.5,
			"Y":  11.5,
			"Y3": 4,
			"B":  0,
		},
		Freqs: map[string]float64{
			"A":  1.0 / 3.0,
			"X":  7.0 / 30.0,
			"X2": infiniteFrequency,
			"Y":  0.4,
			"Y3": infiniteFrequency,
			"B":  0,
		},
		ASet: []*Link{
			allLinks[9], // Y3->B
			allLinks[8], // Y->Y3
			allLinks[7], // X->Y3
			allLinks[6], // Y->B
			allLinks[4], // X2->Y
			allLinks[3], // X->X2
			allLinks[1], // A->X2
			allLinks[0], // A->B
		},
	}
	volumes := AssignDemand(allLinks, allNodes, &optimalStrategy, odMatrix, destinationNode)
	correctVolumes := Volumes{
		Links: map[string]map[string]float64{
			"A": {
				"B":  0.5,
				"X2": 0.5,
			},
			"X2": {
				"X": 0.0,
				"Y": 0.5,
			},
			"X": {
				"X2": 0.0,
				"Y3": 0.0,
			},
			"Y": {
				"Y3": 1.0 / 12.0,
				"B":  5.0 / 12.0,
			},
			"Y3": {
				"Y": 0.0,
				"B": 1.0 / 12.0,
			},
		},
		Nodes: map[string]float64{
			"A":  1.0,
			"X2": 0.5,
			"X":  0.0,
			"Y3": 1.0 / 12.0,
			"Y":  0.5,
			"B":  0.0,
		},
	}
	assert.Equal(t, len(volumes.Links), len(correctVolumes.Links), "Incorrect number of links in volumes data")
	assert.Equal(t, len(volumes.Nodes), len(correctVolumes.Nodes), "Incorrect number of nodes in volumes data")
	const eps = 1e-9
	for fromNode := range volumes.Links {
		assert.Contains(t, correctVolumes.Links, fromNode, "No 'FromNode' in correct volumes data")
		for toNode, volume := range volumes.Links[fromNode] {
			assert.Contains(t, correctVolumes.Links[fromNode], toNode, "No 'ToNode' in correct volumes data")
			assert.InDelta(t, volume, correctVolumes.Links[fromNode][toNode], eps, "Incorrect volume in link (%s, %s)", fromNode, toNode)
		}
	}
	for i, nodeVolume := range volumes.Nodes {
		assert.Contains(t, correctVolumes.Nodes, i, "No node in correct volumes data")
		assert.InDelta(t, nodeVolume, correctVolumes.Nodes[i], eps, "Incorrect volume in node %s", i)
	}
}

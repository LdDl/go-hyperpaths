package hyperpaths

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHyperPaths(t *testing.T) {
	Verbose = true
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
	ops := FindOptimalStrategy(allLinks, allNodes, destinationNode)

	const eps = 1e-9

	expectedLabels := map[string]float64{
		"A":  27.75,
		"X":  19.071428571428573,
		"X2": 17.5,
		"Y":  11.5,
		"Y3": 4,
		"B":  0,
	}
	expectedFreqs := map[string]float64{
		"A":  1.0 / 3.0,
		"X":  7.0 / 30.0,
		"X2": infiniteFrequency,
		"Y":  0.4,
		"Y3": infiniteFrequency,
		"B":  0,
	}
	// Matches the paper order (Spiess & Florian 1989, p. 93-94)
	expectedASet := []*Link{
		allLinks[9], // Y3->B
		allLinks[8], // Y->Y3
		allLinks[7], // X->Y3
		allLinks[6], // Y->B
		allLinks[4], // X2->Y
		allLinks[3], // X->X2
		allLinks[1], // A->X2
		allLinks[0], // A->B
	}

	assert.Equal(t, len(ops.Labels), len(expectedLabels), "Incorrect number of labels")
	assert.Equal(t, len(ops.Freqs), len(expectedFreqs), "Incorrect number of frequencies")
	assert.Equal(t, len(ops.ASet), len(expectedASet), "Incorrect number of links in attractive set")

	for k, v := range ops.Labels {
		assert.Contains(t, expectedLabels, k, "Incorrect label key %s has met", k)
		assert.InDelta(t, v, expectedLabels[k], eps, "Incorrect label value for node %s", k)
	}
	for k, v := range ops.Freqs {
		assert.Contains(t, expectedFreqs, k, "Incorrect frequency key %s has met", k)
		assert.InDelta(t, v, expectedFreqs[k], eps, "Incorrect frequency value for node %s", k)
	}
	for i, v := range ops.ASet {
		fmt.Println(v, expectedASet[i])
		assert.Equal(t, v, expectedASet[i], "Incorrect link in attractive set at index %d", i)
	}
}

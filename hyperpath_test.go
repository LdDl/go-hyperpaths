package hyperpaths

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func paperLinks() []*Link {
	return []*Link{
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
}

func paperNodes() map[string]struct{} {
	return map[string]struct{}{
		"A": {},
		"X": {}, "X2": {},
		"Y": {}, "Y3": {},
		"B": {},
	}
}

func TestHyperPaths(t *testing.T) {
	allNodes := paperNodes()
	allLinks := paperLinks()
	destinationNode := "B"
	ops := FindOptimalStrategy(allLinks, allNodes, destinationNode)

	const eps = 1e-9

	// With exact no-wait handling the labels match the paper exactly:
	// no big-M artifacts like 4.000000000000001
	expectedLabels := map[string]float64{
		"A":  27.75,
		"X":  19.071428571428573,
		"X2": 17.5,
		"Y":  11.5,
		"Y3": 4,
		"B":  0,
	}
	// +Inf marks nodes whose basket is a single no-wait link
	expectedFreqs := map[string]float64{
		"A":  1.0 / 3.0,
		"X":  7.0 / 30.0,
		"X2": math.Inf(+1),
		"Y":  0.4,
		"Y3": math.Inf(+1),
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
		if math.IsInf(expectedFreqs[k], 1) {
			assert.True(t, math.IsInf(v, 1), "Frequency for node %s must be +Inf, got %v", k, v)
		} else {
			assert.InDelta(t, v, expectedFreqs[k], eps, "Incorrect frequency value for node %s", k)
		}
	}
	for i, v := range ops.ASet {
		assert.Equal(t, v, expectedASet[i], "Incorrect link in attractive set at index %d", i)
	}
}

func TestZeroCostChainLoading(t *testing.T) {
	// Regression for the loading order at exact zero-cost ties: the
	// alighting link (B1 -> S2) and the walking link (S2 -> S3) both have
	// key u_j + c_a = 4 and equal tail labels, so no sort comparator can
	// recover the dependency between them. Reverse acceptance order must
	// load the inflow of S2 before its outflow.
	allNodes := map[string]struct{}{
		"S1": {}, "B0": {}, "B1": {}, "S2": {}, "S3": {},
	}
	allLinks := []*Link{
		// boarding: wait for the bus (headway 5), no riding yet
		{"S1", "B0", "Bus", 0, 5},
		// on-board segment
		{"B0", "B1", "Bus", 10, 0},
		// alighting, key u_S2 + 0 = 4
		{"B1", "S2", "Bus", 0, 0},
		// walking to the destination, key u_S3 + 4 = 4
		{"S2", "S3", "Walk", 4, 0},
	}
	ops := FindOptimalStrategy(allLinks, allNodes, "S3")
	// 5 wait + 10 ride + 0 alight + 4 walk
	assert.InDelta(t, ops.Labels["S1"], 19.0, 1e-12)

	trips := map[string]map[string]float64{
		"S1": {"S3": 100},
	}
	volumes := AssignDemand(allLinks, allNodes, ops, trips, "S3")
	assert.InDelta(t, volumes.Links["S1"]["B0"], 100.0, 1e-12)
	assert.InDelta(t, volumes.Links["B0"]["B1"], 100.0, 1e-12)
	assert.InDelta(t, volumes.Links["B1"]["S2"], 100.0, 1e-12)
	assert.InDelta(t, volumes.Links["S2"]["S3"], 100.0, 1e-12)
}

func TestNoWaitReplacesBasket(t *testing.T) {
	// A boarding link enters the basket of I first (key 4), then a
	// cheaper no-wait chain I->W->D (key 5 < current u_I = 10) must
	// replace it entirely: exact label, infinite frequency, single link.
	allNodes := map[string]struct{}{
		"I": {}, "W": {}, "D": {},
	}
	allLinks := []*Link{
		// boarding link, key u_D + 4 = 4, accepted first: u_I = 6 + 4 = 10
		{"I", "D", "Bus", 4, 6},
		// no-wait walk, key u_W + 3 = 5, replaces the basket: u_I = 5
		{"I", "W", "Walk", 3, 0},
		// no-wait walk, key 2
		{"W", "D", "Walk", 2, 0},
	}
	ops := FindOptimalStrategy(allLinks, allNodes, "D")

	assert.InDelta(t, ops.Labels["I"], 5.0, 1e-12)
	assert.InDelta(t, ops.Labels["W"], 2.0, 1e-12)
	assert.True(t, math.IsInf(ops.Freqs["I"], 1))
	assert.True(t, math.IsInf(ops.Freqs["W"], 1))

	// The replaced boarding link I->D must not remain attractive
	assert.Equal(t, 2, len(ops.ASet), "basket of I must hold only the no-wait link")
	for _, link := range ops.ASet {
		assert.Equal(t, float64(0), link.Headway, "only no-wait links expected in the attractive set")
	}

	// Loading: the single no-wait link takes the whole node volume
	trips := map[string]map[string]float64{
		"I": {"D": 1},
	}
	volumes := AssignDemand(allLinks, allNodes, ops, trips, "D")
	assert.InDelta(t, volumes.Links["I"]["W"], 1.0, 1e-12)
	assert.InDelta(t, volumes.Links["W"]["D"], 1.0, 1e-12)
	assert.InDelta(t, volumes.Links["I"]["D"], 0.0, 1e-12)
}

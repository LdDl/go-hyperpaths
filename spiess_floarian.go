package hyperpaths

// SFResult is the result of running through the Spiess-Florian algorithm
type SFResult struct {
	// Optimal strategy
	Strategy *Strategy
	// Assigned demand
	Volumes *Volumes
}

// ComputeSF computes the Spiess-Florian algorithm
func ComputeSF(allLinks []*Link, allStops map[string]struct{}, destination string, odMatrix map[string]map[string]float64) *SFResult {
	// Part 1: Find optimal strategy
	ops := FindOptimalStrategy(allLinks, allStops, destination)
	// Part 2: Assign demand according to optimal strategy
	volumes := AssignDemand(allLinks, allStops, ops, odMatrix, destination)
	return &SFResult{ops, volumes}
}

package hyperpaths

import (
	"fmt"
	"math"
)

// Strategy is the optimal strategy as defined in the Spiess-Florian algorithm.
type Strategy struct {
	// u_{i} - expected travel time from node i to destination
	Labels map[string]float64
	// f_{i} - combined frequency of attractive links at node i.
	// +Inf marks a node whose basket is a single no-wait link.
	Freqs map[string]float64
	// \overline{A} - attractive links forming the hyperpath
	ASet []*Link
}

const (
	// The waiting-time constant of the Spiess-Florian expected travel time:
	//   u_i = (1 + sum(f_a * (c_a + u_j))) / f_i
	// When the first attractive link arrives at a node the sum is empty and
	// the numerator starts from this constant.
	alpha = 1.0
)

var (
	Verbose = false
)

func FindOptimalStrategy(allLinks []*Link, allStops map[string]struct{}, destination string) *Strategy {
	/* 1.1 Initialization */
	if Verbose {
		fmt.Println("1.1 Initialization \\\\")
	}

	// Integer arena: map every node name to a dense int index once, so the
	// hot loops below index slices instead of hashing strings on every
	// access. allStops is interned first (indices [0, nStops)) so the
	// returned Labels/Freqs keep exactly the allStops key set; any link
	// endpoint outside allStops (out of contract) is appended after and left
	// out of the result.
	nID := make(map[string]int, len(allStops))
	nName := make([]string, 0, len(allStops))
	intern := func(s string) int {
		if id, ok := nID[s]; ok {
			return id
		}
		id := len(nName)
		nID[s] = id
		nName = append(nName, s)
		return id
	}
	for stop := range allStops {
		intern(stop)
	}
	nStops := len(nName)

	m := len(allLinks)
	lFrom := make([]int, m)
	lTo := make([]int, m)
	lCost := make([]float64, m)
	lHead := make([]float64, m)
	for k, link := range allLinks {
		lFrom[k] = intern(link.FromNode)
		lTo[k] = intern(link.ToNode)
		lCost[k] = link.TravelCost
		lHead[k] = link.Headway
	}
	destID := intern(destination)
	n := len(nName)

	// u_i and f_i as dense slices instead of map[string]float64.
	u := make([]float64, n)
	f := make([]float64, n)
	for id := 0; id < n; id++ {
		f[id] = 0.0
		if id == destID {
			u[id] = 0.0
		} else {
			u[id] = math.Inf(+1)
		}
	}
	if Verbose {
		for id := 0; id < n; id++ {
			fmt.Printf("$f_{%s} = 0$ \\\\ \n", nName[id])
			if id == destID {
				fmt.Printf("$u_{%s} = 0$ \\\\ \n", nName[id])
			} else {
				fmt.Printf("$u_{%s} = Infinity$ \\\\ \n", nName[id])
			}
		}
	}

	// overlineA holds accepted link indices in acceptance order; a no-wait
	// link replaces a node's whole basket, and replaced slots become -1 and
	// are compacted at the end. aSetIdx[node] are that node's positions in
	// overlineA.
	overlineA := make([]int, 0, m/2)
	aSetIdx := make([][]int, n)

	// Adjacency by head node: the link indices whose ToNode == node, so that
	// when u[node] improves exactly those incoming links are re-keyed.
	adjByTo := make([][]int, n)
	for k := 0; k < m; k++ {
		adjByTo[lTo[k]] = append(adjByTo[lTo[k]], k)
	}

	// One priority-queue entry per link, in a single backing slab (one
	// allocation for all m entries instead of m); &slab[k] reaches a link's
	// entry directly in the update step, with no scan.
	slab := make([]pqEntry, m)
	pq := make(PriorityQueue, 0, m)
	for k := 0; k < m; k++ {
		slab[k] = pqEntry{
			link:     k,
			priority: u[lTo[k]] + lCost[k],
		}
		pq = append(pq, &slab[k])
	}
	pq.Init()
	if Verbose {
		pq.Print()
	}
	for pq.Len() > 0 {
		/* 1.2 Get next link */
		if Verbose {
			pq.Print()
		}
		entry := pq.Pop().(*pqEntry)
		if math.IsInf(entry.priority, 1) {
			break
		}
		k := entry.link
		i := lFrom[k]
		j := lTo[k]
		sumUC := u[j] + lCost[k]

		/* 1.3 Update node label */
		if Verbose {
			fmt.Printf("Process: $a = (i, j) = (%s, %s)$, \\\\ \n", nName[i], nName[j])
		}
		// A node already served by a no-wait link is final: the no-wait
		// link absorbs all flow (its share f_a/f_i is 1 in the limit),
		// so no other link may enter the basket
		if math.IsInf(f[i], 1) {
			continue
		}
		// Strict improvement test: a link is accepted only if it
		// strictly improves the label. Step 1.3 of Spiess & Florian
		// (1989) prints the nonstrict u_i >= u_j + c_a, but the two
		// rules differ only at exact equality, where the update is a
		// no-op (the combination formula returns u_i unchanged; for
		// f_a = inf the basket is replaced at the same value): labels,
		// expected travel times and every number published in the
		// paper are identical either way. The strict form is what
		// part 2 needs. Step 2.2 loads links "in reverse topological
		// order (decreasing u_j + c_a)" (p. 94) and Proposition 4
		// claims flow conservation "by construction" - both presume an
		// acyclic strategy, which the nonstrict rule does not
		// guarantee: in an expanded route graph a boarding link (cost
		// 0) into a route node whose label came from its own alighting
		// link (cost 0) has key exactly u_i, so >= admits a zero-cost
		// stop -> node -> stop cycle and the one-pass loading strands
		// the volume entering it (see
		// TestBoardAlightLoopConservation). Rejecting at equality
		// keeps the strategy acyclic and stays optimal: for the
		// rejected link mu_a = 0 satisfies dual feasibility (20) as an
		// equality and complementary slackness (24) holds since
		// v_a = 0, a degenerate optimum. The prose of p. 94 ("if this
		// time is smaller than u_i, link a is included") describes
		// exactly this strict rule. All step, equation and page
		// references above are to the original paper, not to the
		// spiess_floarian.tex excerpt in this repository.
		if u[i] <= sumUC {
			continue
		}
		if Verbose {
			fmt.Printf("\\quad $u_i \\leq u_j + c_a : %v \\leq %v + %v$ - FALSE \\\\ \n", u[i], u[j], lCost[k])
		}
		if lHead[k] <= 0 {
			// No-wait link (infinite frequency): the modified step 1.3
			// given by the paper on p. 96 - the exact limit of the label
			// update formula as f_a -> inf. The link replaces the whole
			// attractive basket:
			//   u_i := u_j + c_a, f_i := inf, A_i := {a}
			u[i] = sumUC
			f[i] = math.Inf(+1)
			for _, idx := range aSetIdx[i] {
				overlineA[idx] = -1
			}
			aSetIdx[i] = aSetIdx[i][:0]
			overlineA = append(overlineA, k)
			aSetIdx[i] = append(aSetIdx[i], len(overlineA)-1)
			if Verbose {
				fmt.Printf("\\quad no-wait link: $u_i = u_j + c_a = %v$, $f_i = \\infty$, basket replaced by $(%s, %s)$ \\\\ \n", u[i], nName[i], nName[j])
			}
		} else {
			freq := 1.0 / lHead[k]
			if Verbose {
				fmt.Printf("\\quad $f_a = %v$ \\\\ \n", freq)
				fmt.Printf("\\quad $u_j + c_a = %v$ \\\\ \n", sumUC)
				fmt.Printf("\\quad $u_i = %v$ \\\\ \n", u[i])
			}
			if f[i] == 0 {
				// First link in the basket: u_i = (1 + f_a*(u_j+c_a)) / f_a
				u[i] = (alpha + freq*sumUC) / freq
			} else {
				u[i] = (f[i]*u[i] + freq*sumUC) / (f[i] + freq)
			}
			f[i] += freq
			overlineA = append(overlineA, k)
			aSetIdx[i] = append(aSetIdx[i], len(overlineA)-1)
			if Verbose {
				fmt.Printf("\\quad$u_i = \\frac{f_i * u_i + f_a * (u_j + c_a)}{f_i + f_a} = %v$, $f_i = %v$ \\\\ \n", u[i], f[i])
				fmt.Printf("\\quad $\\overline{A} = \\overline{A} \\cup {(%s, %s)}$ \\\\ \n", nName[i], nName[j])
			}
		}

		// u[i] improved: re-key exactly the links entering i.
		for _, kk := range adjByTo[i] {
			pq.update(&slab[kk], u[i]+lCost[kk])
		}
		if Verbose {
			fmt.Println("Node labels: \\\\")
			for id := 0; id < n; id++ {
				fmt.Printf("$%s -> (u_i, f_i) = (%v, %v)$ \\\\ \n", nName[id], u[id], f[id])
			}
		}
	}

	// Compact the attractive set: drop entries replaced by no-wait links.
	// The append order is preserved, i.e. non-decreasing u_j + c_a.
	aSet := make([]*Link, 0, len(overlineA))
	for _, k := range overlineA {
		if k >= 0 {
			aSet = append(aSet, allLinks[k])
		}
	}

	// Translate the arena labels/freqs back to the public string-keyed maps,
	// for the allStops key set only.
	labels := make(map[string]float64, nStops)
	freqs := make(map[string]float64, nStops)
	for id := 0; id < nStops; id++ {
		labels[nName[id]] = u[id]
		freqs[nName[id]] = f[id]
	}

	strategy := &Strategy{
		Labels: labels,
		Freqs:  freqs,
		ASet:   aSet,
	}
	return strategy
}

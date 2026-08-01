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
	u := make(map[string]float64, len(allStops))
	f := make(map[string]float64, len(allStops))
	for stop := range allStops {
		if Verbose {
			fmt.Printf("$f_{%s} = 0$ \\\\ \n", stop)
		}
		f[stop] = 0.0
		if stop == destination {
			if Verbose {
				fmt.Printf("$u_{%s} = 0$ \\\\ \n", destination)
			}
			u[stop] = 0.0
			continue
		}
		if Verbose {
			fmt.Printf("$u_{%s} = Infinity$ \\\\ \n", stop)
		}
		u[stop] = math.Inf(+1)
	}

	overlineA := make([]*Link, 0, len(allLinks)/2)
	// Positions of each node's basket links inside overlineA, so that a
	// no-wait link can replace the whole basket. Replaced entries are set
	// to nil and compacted at the end.
	aSetIdx := make(map[string][]int)

	linksByToNode := make(map[string][]*Link)
	for _, link := range allLinks {
		linksByToNode[link.ToNode] = append(linksByToNode[link.ToNode], link)
	}

	entries := make(map[string][]*pqEntry, len(allLinks))
	pq := make(PriorityQueue, 0, len(allLinks))
	for _, link := range allLinks {
		entry := &pqEntry{
			link:     link,
			priority: u[link.ToNode] + link.TravelCost,
		}
		entries[link.FromNode] = append(entries[link.FromNode], entry)
		pq = append(pq, entry)
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
		a := entry.link
		i := a.FromNode
		j := a.ToNode
		sumUC := u[j] + a.TravelCost

		/* 1.3 Update node label */
		if Verbose {
			fmt.Printf("Process: $a = (i, j) = (%s, %s)$, \\\\ \n", i, j)
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
			fmt.Printf("\\quad $u_i \\leq u_j + c_a : %v \\leq %v + %v$ - FALSE \\\\ \n", u[i], u[j], a.TravelCost)
		}
		if a.Headway <= 0 {
			// No-wait link (infinite frequency): the modified step 1.3
			// given by the paper on p. 96 - the exact limit of the label
			// update formula as f_a -> inf. The link replaces the whole
			// attractive basket:
			//   u_i := u_j + c_a, f_i := inf, A_i := {a}
			u[i] = sumUC
			f[i] = math.Inf(+1)
			for _, idx := range aSetIdx[i] {
				overlineA[idx] = nil
			}
			aSetIdx[i] = aSetIdx[i][:0]
			overlineA = append(overlineA, a)
			aSetIdx[i] = append(aSetIdx[i], len(overlineA)-1)
			if Verbose {
				fmt.Printf("\\quad no-wait link: $u_i = u_j + c_a = %v$, $f_i = \\infty$, basket replaced by $(%s, %s)$ \\\\ \n", u[i], i, j)
			}
		} else {
			freq := 1.0 / a.Headway
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
			overlineA = append(overlineA, a)
			aSetIdx[i] = append(aSetIdx[i], len(overlineA)-1)
			if Verbose {
				fmt.Printf("\\quad$u_i = \\frac{f_i * u_i + f_a * (u_j + c_a)}{f_i + f_a} = %v$, $f_i = %v$ \\\\ \n", u[i], f[i])
				fmt.Printf("\\quad $\\overline{A} = \\overline{A} \\cup {(%s, %s)}$ \\\\ \n", i, j)
			}
		}

		if linksToUpdate, exists := linksByToNode[i]; exists {
			for _, link := range linksToUpdate {
				if iEntries, hasEntries := entries[link.FromNode]; hasEntries {
					for _, entry := range iEntries {
						if entry.link.ToNode == i && entry.link.FromNode == link.FromNode {
							pq.update(entry, u[i]+link.TravelCost)
							break
						}
					}
				}
			}
		}
		if Verbose {
			fmt.Println("Node labels: \\\\")
			for s := range allStops {
				fmt.Printf("$%s -> (u_i, f_i) = (%v, %v)$ \\\\ \n", s, u[s], f[s])
			}
		}
	}

	// Compact the attractive set: drop entries replaced by no-wait links.
	// The append order is preserved, i.e. non-decreasing u_j + c_a.
	aSet := make([]*Link, 0, len(overlineA))
	for _, link := range overlineA {
		if link != nil {
			aSet = append(aSet, link)
		}
	}

	strategy := &Strategy{
		Labels: u,
		Freqs:  f,
		ASet:   aSet,
	}
	return strategy
}

// Implementation of the Spiess-Florian algorithm for transit assignment. See the ref. at spiess_floarian.tex LaTeX file.

package hyperpaths

import (
	"fmt"
	"math"
)

// Strategy is optimal strategy as it defined in Spiess-Florian algorithm
type Strategy struct {
	// u_{i} - expected time to destination
	Labels map[string]float32
	// f_{i} - combined frequency at node
	Freqs map[string]float32
	// \overline{A} - attractive links in hyperpath
	ASet []*Link
}

const (
	ALPHA             = float32(1.0)
	infiniteFrequency = float32(99999999999.0)
)

var (
	Verbose    = false
	mathINFf32 = float32(math.Inf(+1))
)

func FindOptimalStrategy(allLinks []*Link, allStops map[string]struct{}, destination string) *Strategy {
	/* 1.1 Initialization */
	if Verbose {
		fmt.Println("1.1 Initialization \\\\")
	}
	u := make(map[string]float32, len(allStops))
	f := make(map[string]float32, len(allStops))
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
		u[stop] = mathINFf32
	}

	// Attractive set
	overlineA := make([]*Link, 0, len(allLinks)/2) // Just prealloc some capacity

	// Precompute a map of links by ToNode for faster updates
	linksByToNode := make(map[string][]*Link)
	for _, link := range allLinks {
		linksByToNode[link.ToNode] = append(linksByToNode[link.ToNode], link)
	}

	// Build priority queue (S - active links)
	// Track entries by FromNode for quick updates
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

	for pq.Len() > 0 {
		/* 1.2 Get next link */
		entry := pq.Pop().(*pqEntry)
		if math.IsInf(float64(entry.priority), 1) || entry.priority >= mathINFf32 {
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
		// Skip if not improving current label
		if u[i] < sumUC {
			// fmt.Printf("\\quad $u_i < u_j + c_a : %v < %v + %v$ - TRUE \\\\ \n", u[i], u[j], a.TravelCost)
			continue
		}
		if Verbose {
			fmt.Printf("\\quad $u_i < u_j + c_a : %v < %v + %v$ - FALSE \\\\ \n", u[i], u[j], a.TravelCost)
		}
		// Calculate frequency (1/headway)
		freq := infiniteFrequency
		if a.Headway > 0 {
			freq = 1 / a.Headway
		}
		if Verbose {
			fmt.Printf("\\quad $f_a = %v$ \\\\ \n", freq)
			fmt.Printf("\\quad $u_j + c_a = %v$ \\\\ \n", u[j]+a.TravelCost)
			fmt.Printf("\\quad $u_i = %v$ \\\\ \n", u[i])
			fmt.Printf("\\quad$u_i = \\frac{f_i * u_i + f_a * (u_j + c_a)}{f_i + f_a} = \\frac{(%v) * (%v) + (%v) * ((%v) + (%v))}{(%v) + (%v)} = $ \\\\ \n",
				f[i], u[i], freq, u[j], a.TravelCost, f[i], freq,
			)
		}
		numeratorPart := f[i] * u[i]
		if math.IsNaN(float64(numeratorPart)) {
			numeratorPart = ALPHA
		}
		numeratorPart2 := freq * (u[j] + a.TravelCost)
		if math.IsNaN(float64(numeratorPart2)) {
			numeratorPart2 = ALPHA
		}
		numerator := numeratorPart + numeratorPart2
		denominator := f[i] + freq
		u[i] = numerator / denominator
		if Verbose {
			fmt.Printf("\\quad \\quad $\\frac{(%v) + (%v)}{(%v) + (%v)} = \\frac{%v}{%v} = %v$ \\\\ \n", numeratorPart, numeratorPart2, f[i], freq, numerator, denominator, u[i])
			fmt.Printf("\\quad $f_i = f_{i} + f_a = (%v) + (%v) = %v$ \\\\ \n", f[i], freq, denominator)
			fmt.Printf("\\quad $\\overline{A} = \\overline{A} \\cup {a} = \\overline{A} \\cup {(%s, %s)}$ \\\\ \n", i, j)
		}
		f[i] = denominator

		// Update attractive set
		overlineA = append(overlineA, a)

		// Update priority queue (for u[i]) - OPTIMIZED VERSION
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
	strategy := &Strategy{
		Labels: u,
		Freqs:  f,
		ASet:   overlineA,
	}
	return strategy
}

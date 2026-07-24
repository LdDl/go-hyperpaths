package hyperpaths

import (
	"fmt"
	"math"
)

// Strategy is the optimal strategy as defined in the Spiess-Florian algorithm.
type Strategy struct {
	// u_{i} - expected travel time from node i to destination
	Labels map[string]float64
	// f_{i} - combined frequency of attractive links at node i
	Freqs map[string]float64
	// \overline{A} - attractive links forming the hyperpath
	ASet []*Link
}

const (
	// When the first attractive link arrives at a node, f_i = 0 and u_i = +Inf,
	// so f_i * u_i = 0 * Inf = NaN in IEEE 754. The correct mathematical value
	// is 1: the Spiess-Florian expected travel time is
	//   u_i = (1 + sum(f_a * (c_a + u_j))) / f_i
	// so the product f_i * u_i = 1 + sum(...). At initialization the sum is
	// empty, leaving f_i * u_i = 1. This constant replaces the NaN.
	alpha = 1.0

	// Frequency used for on-board (riding) links where headway = 0.
	// Must be finite to avoid Inf * 0 = NaN in the update formula.
	// 1e15 gives an effective wait of 1e-15 time units - negligible.
	infiniteFrequency = 1e15
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
		if u[i] < sumUC {
			continue
		}
		if Verbose {
			fmt.Printf("\\quad $u_i < u_j + c_a : %v < %v + %v$ - FALSE \\\\ \n", u[i], u[j], a.TravelCost)
		}
		freq := infiniteFrequency
		if a.Headway > 0 {
			freq = 1.0 / a.Headway
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
		if math.IsNaN(numeratorPart) {
			numeratorPart = alpha
		}
		numeratorPart2 := freq * (u[j] + a.TravelCost)
		if math.IsNaN(numeratorPart2) {
			numeratorPart2 = alpha
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

		overlineA = append(overlineA, a)

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

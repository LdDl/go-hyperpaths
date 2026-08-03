# Hyperpath routing in Golang

Just implementation of [Spiess, H. and Florian, M. (1989) "Optimal strategies: A new assignment model for transit networks"](https://doi.org/10.1016/0191-2615(89)90034-9) in Golang

## Algorithm

Here is copy of algorithm in MathJax (for the LaTeX see [spiess_floarian.tex](./spiess_floarian.tex)):

### Part 1: Find optimal strategy

1. **Initialization**
   - Set $u_r = 0$ for destination node
   - Set $u_i = \infty$ for all other nodes
   - Set $f_i = 0$ for all nodes
   - Initialize empty attractive set $\overline{A}$

2. **Label Setting** (repeated until the link set is exhausted)
   - Take the unexamined link $a = (i,j)$ with minimum $u_j + c_a$
   - If $u_i \geq u_j + c_a$:
     * Update node label: $$u_i = \frac{f_i \cdot u_i + f_a \cdot (u_j + c_a)}{f_i + f_a}$$
     * Update frequency: $$f_i = f_i + f_a$$
     * Add to attractive set: $$\overline{A} = \overline{A} \cup \{a\}$$

   Note: the implementation accepts a link only on strict improvement,
   $u_i > u_j + c_a$. At exact equality the update above is a no-op, so
   labels and costs are identical, but accepting such links can close
   zero-cost board-alight cycles on which the one-pass loading of part 2
   loses flow. See the remark in [spiess_floarian.tex](./spiess_floarian.tex)
   and `TestBoardAlightLoopConservation`.

### Part 2: Assign demand according to optimal strategy

1. **Initialization**
   - Set $V_i = g_i$ for all nodes

2. **Loading**
   - Process links in decreasing order of $u_j + c_a$
   - For attractive links $a \in \overline{A}$:
     * Calculate volume: $$v_a = \frac{f_a}{f_i}V_i$$
     * Update node volume: $$V_j = V_j + v_a$$

### Infinite frequencies (no-wait links)

Links with `Headway = 0` (walking, alighting, on-board) have infinite frequency. Instead of a big-M constant, the implementation follows the modified version of the algorithm given by the paper itself (p. 96): such a link replaces the whole attractive set of its tail node, $u_i := u_j + c_a$, $f_i := \infty$, $\overline{A}_i := \{a\}$, and during loading it takes the entire node volume ($v_a := V_i$). The paper's own worked example uses this modified version, so the labels match it exactly (e.g. $u_{Y3} = 4$, not $4 + \varepsilon$).

The loading phase needs no sorting, as the paper notes on p. 97: "no additional computations are needed to establish the order in which the links are processed, since it is the inverse of the order used in part 1 of the algorithm". The attractive set is built in acceptance order (non-decreasing $u_j + c_a$), so reverse iteration is exactly the required decreasing order (Table 3 of the paper), and at zero-cost ties it loads a node's inflow links before its outflow links by construction.

## How to use

* Get the package:
   ```shell
   go get github.com/lddl/go-hyperpaths
   ```

* Code (you can find it in [examples/paper](./examples/paper))
   ```go
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
   ```

## Two ways to use it

Both APIs run the same algorithm and give identical results; pick by use case.

### Simple (string) API - for debugging and single solves

`ComputeSF` / `FindOptimalStrategy` / `AssignDemand` (shown above) take string node names and a `map[origin]map[dest]float64` OD, and return human-readable maps. It is the easiest to read, and it is the debugging path: set `hyperpaths.Verbose = true` to print a step-by-step trace of both phases. Internally it already uses the integer arena, so a single solve is fast.

Use it for: one-off / single-destination solves, small networks, debugging, first integration.

### Arena API - for many destinations and long-running / multi-request services

Intern the network once into a `Graph`, then reuse a `Workspace` across destinations. Results are integer-indexed (arena) and, once the workspace is warm, allocation-free. On a full assignment (every stop a destination) this is about an order of magnitude faster than calling `ComputeSF` per destination.

```go
// once; immutable, shareable
g := hyperpaths.NewGraph(allLinks, allNodes)
// reusable buffers
w := g.NewWorkspace()

// od is map[origin]map[dest]float64
w.SolveEach(od, func(res *hyperpaths.DestResult) {
    // res.Labels[i], res.LinkVol[k], res.NodeVol[i] are arena-indexed;
    // use g.NodeName(i) / g.NodeIndex(name) to translate to/from names.
    // res is reused on the next destination - copy out what you keep.
})
```

For a single destination with an integer demand column there is also the lower-level `w.Assign(destID, demand)`.

Use it for: assigning many destinations, large networks, long-running or multi-user services.

### Concurrency

A `Graph` is immutable and safe to share across goroutines; a `Workspace` is mutable and must not be shared. Build the graph once at startup and pool the workspaces, one per in-flight request:

```go
graph := hyperpaths.NewGraph(links, stops)
pool := sync.Pool{New: func() any { return graph.NewWorkspace() }}
// per request (own goroutine):
w := pool.Get().(*hyperpaths.Workspace)
defer pool.Put(w)
w.SolveEach(od, func(res *hyperpaths.DestResult) {
   // accumulate res.LinkVol
})
```

Do not mutate a shared network, and do not toggle `Verbose`, while assignments are running.

## References
Spiess, H. and Florian, M. (1989) "Optimal strategies: A new assignment model for transit networks". Transportation Research Part B: Methodological, 23(2), 83-102. Available in: https://doi.org/10.1016/0191-2615(89)90034-9
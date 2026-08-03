package hyperpaths

import "fmt"

// ExampleGraph shows the arena Graph/Workspace API: build the graph once,
// reuse a Workspace, and assign a full OD with SolveEach.
func ExampleGraph() {
	links := paperLinks()
	nodes := paperNodes()

	g := NewGraph(links, nodes)
	w := g.NewWorkspace()

	// One trip from A to B.
	od := map[string]map[string]float64{"A": {"B": 1.0}}
	aID := g.NodeIndex("A")

	w.SolveEach(od, func(res *DestResult) {
		fmt.Printf("A -> B expected time: %.2f\n", res.Labels[aID])
	})
	// Output:
	// A -> B expected time: 27.75
}

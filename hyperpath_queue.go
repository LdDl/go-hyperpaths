package hyperpaths

type pqEntry struct {
	link     *Link
	priority float32 // u_j + c_a (used for prioritization)
	index    int     // The index of the item in the heap.
}

type PriorityQueue []*pqEntry

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].priority <= pq[j].priority }
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*pqEntry)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	n := len(*pq)
	if n == 0 {
		return nil
	}
	old := *pq
	first := old[0]
	last := old[n-1]
	old[0] = last
	last.index = 0
	*pq = old[0 : n-1]
	pq.siftDown(0)
	first.index = -1
	return first
}

func (pq PriorityQueue) Init() {
	half := len(pq) / 2
	for i := half - 1; i >= 0; i-- {
		pq[i].index = i
		pq[i+half].index = i + half
		pq.siftDown(i)
	}
}

// update modifies the priority and value of an pqEntry in the queue.
func (pq *PriorityQueue) update(item *pqEntry, priority float32) {
	oldPriority := item.priority
	item.priority = priority

	// If priority decreased (higher priority in min-heap), sift up
	if priority <= oldPriority {
		pq.siftUp(item.index)
	} else {
		// If priority increased (lower priority in min-heap), sift down
		pq.siftDown(item.index)
	}
}

// siftUp moves an element up the heap until heap property is satisfied
func (pq *PriorityQueue) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if pq.Less(i, parent) {
			pq.Swap(i, parent)
			i = parent
		} else {
			break
		}
	}
}

// siftDown moves an element down the heap until heap property is satisfied
func (pq *PriorityQueue) siftDown(i int) {
	n := pq.Len()
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2

		if left < n && pq.Less(left, smallest) {
			smallest = left
		}
		if right < n && pq.Less(right, smallest) {
			smallest = right
		}

		if smallest != i {
			pq.Swap(i, smallest)
			i = smallest
		} else {
			break
		}
	}
}

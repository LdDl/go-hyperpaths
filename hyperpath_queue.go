package hyperpaths

import (
	"fmt"
	"strings"
)

type pqEntry struct {
	link     *Link
	priority float64
	index    int
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
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

func (pq PriorityQueue) Init() {
	for i := range pq {
		pq[i].index = i
	}
	for i := len(pq)/2 - 1; i >= 0; i-- {
		pq.siftDown(i)
	}
}

func (pq *PriorityQueue) update(item *pqEntry, priority float64) {
	if item.index < 0 {
		return
	}
	oldPriority := item.priority
	item.priority = priority
	if priority <= oldPriority {
		pq.siftUp(item.index)
	} else {
		pq.siftDown(item.index)
	}
}

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

func (pq PriorityQueue) Print() {
	if pq.Len() == 0 {
		fmt.Println("Priority Queue: <empty>")
		return
	}
	arr := make([]string, pq.Len())
	for i, entry := range pq {
		arr[i] = "(" + entry.link.FromNode + "," + entry.link.ToNode + ") == " + fmt.Sprintf("%.2f", entry.priority)
	}
	fmt.Printf("Priority Queue: [%s]\\\\ \n", strings.Join(arr, ", "))
}

package pq

type Comparator[T any] func(a, b T) bool

type Item[T any] struct {
	Value T
	index int
}

// Index returns the current index of the item in the heap.
func (item *Item[T]) Index() int {
	return item.index
}

// PriorityQueue is a generic priority queue implementation using a binary heap data structure.
type PriorityQueue[T any] struct {
	data []*Item[T]
	cmp  Comparator[T]
}

// New creates a new priority queue.
// cmp: Comparison function to determine element priority.
// Returns: Pointer to the initialized priority queue.
func New[T any](cmp Comparator[T]) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		data: make([]*Item[T], 0),
		cmp:  cmp,
	}
}

// NewPriorityQueue is an alias for New.
func NewPriorityQueue[T any](cmp Comparator[T]) *PriorityQueue[T] {
	return New[T](cmp)
}

// Len returns the number of elements in the queue.
// Returns: The length of the queue.
func (pq *PriorityQueue[T]) Len() int {
	return len(pq.data)
}

// Empty checks if the queue is empty.
// Returns: true if the queue is empty, false otherwise.
func (pq *PriorityQueue[T]) Empty() bool {
	return len(pq.data) == 0
}

// Peek returns the highest priority item without removing it from the queue.
// Returns: The highest priority item, or nil if the queue is empty.
func (pq *PriorityQueue[T]) Peek() *Item[T] {
	if len(pq.data) == 0 {
		return nil
	}
	return pq.data[0]
}

// Push adds a new element to the queue.
// v: The value to add.
// Returns: Pointer to the newly created Item.
func (pq *PriorityQueue[T]) Push(v T) *Item[T] {
	item := &Item[T]{Value: v}
	item.index = len(pq.data)

	pq.data = append(pq.data, item)
	pq.shiftUp(item.index)

	return item
}

// Pop removes and returns the highest priority element from the queue.
// Returns: The highest priority element, or nil if the queue is empty.
func (pq *PriorityQueue[T]) Pop() *Item[T] {
	n := len(pq.data) - 1
	if n < 0 {
		return nil
	}

	pq.swap(0, n)

	item := pq.data[n]
	pq.data = pq.data[:n]

	if len(pq.data) > 0 {
		pq.shiftDown(0)
	}

	item.index = -1
	return item
}

// Remove removes the specified item from the queue.
// item: The item to remove.
func (pq *PriorityQueue[T]) Remove(item *Item[T]) {
	i := item.index
	last := len(pq.data) - 1

	if i == last {
		pq.data = pq.data[:last]
		item.index = -1
		return
	}

	pq.swap(i, last)

	pq.data = pq.data[:last]

	pq.fix(i)

	item.index = -1
}

// Update updates the value of the specified item and re-adjusts the queue structure.
// item: The item to update.
// v: The new value.
func (pq *PriorityQueue[T]) Update(item *Item[T], v T) {
	item.Value = v
	pq.fix(item.index)
}

// Build builds the queue from a slice in O(n) time complexity.
// items: Initial slice of elements.
func (pq *PriorityQueue[T]) Build(items []T) {
	pq.data = make([]*Item[T], len(items))

	for i, v := range items {
		pq.data[i] = &Item[T]{
			Value: v,
			index: i,
		}
	}

	for i := len(pq.data)/2 - 1; i >= 0; i-- {
		pq.shiftDown(i)
	}
}

// fix fixes the heap structure at the specified position by shifting up or down.
// i: Index of the position to fix.
func (pq *PriorityQueue[T]) fix(i int) {
	if !pq.shiftUp(i) {
		pq.shiftDown(i)
	}
}

// shiftUp moves the element at the specified position upward to maintain heap property.
// i: Starting index.
// Returns: true if the element was moved, false otherwise.
func (pq *PriorityQueue[T]) shiftUp(i int) bool {
	moved := false

	for i > 0 {
		parent := (i - 1) / 2

		if !pq.cmp(pq.data[i].Value, pq.data[parent].Value) {
			break
		}

		pq.swap(i, parent)
		i = parent
		moved = true
	}

	return moved
}

// shiftDown moves the element at the specified position downward to maintain heap property.
// i: Starting index.
func (pq *PriorityQueue[T]) shiftDown(i int) {
	n := len(pq.data)

	for {
		left := 2*i + 1
		right := left + 1
		best := i

		if left < n && pq.cmp(pq.data[left].Value, pq.data[best].Value) {
			best = left
		}

		if right < n && pq.cmp(pq.data[right].Value, pq.data[best].Value) {
			best = right
		}

		if best == i {
			return
		}

		pq.swap(i, best)
		i = best
	}
}

// swap swaps two elements at the specified positions and updates their indices.
// i, j: Indices of the positions to swap.
func (pq *PriorityQueue[T]) swap(i, j int) {
	pq.data[i], pq.data[j] = pq.data[j], pq.data[i]

	pq.data[i].index = i
	pq.data[j].index = j
}

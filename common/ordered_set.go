package common

import (
	"errors"
	"sync"
)

// OrderedSet is a set with limited capacity.
// Items are evicted according to their insertion order.
type OrderedSet struct {
	capacity int
	set      map[interface{}]struct{}
	queue    []interface{}
	start    int
	end      int

	lock sync.RWMutex
}

// NewOrderedSet creates an ordered set with given capacity
func NewOrderedSet(capacity int) (*OrderedSet, error) {
	if capacity < 1 {
		return nil, errors.New("capacity must be a positive integer")
	}

	return &OrderedSet{
		capacity: capacity,
		set:      map[interface{}]struct{}{},
		queue:    make([]interface{}, capacity),
		end:      -1,
	}, nil
}

// Add inserts items into the set.
// If capacity is reached, oldest items are evicted
func (os *OrderedSet) Add(items ...interface{}) {
	os.lock.Lock()
	defer os.lock.Unlock()

	for _, item := range items {
		if _, ok := os.set[item]; ok {
			continue
		}

		next := (os.end + 1) % os.capacity
		if os.end != -1 && next == os.start {
			delete(os.set, os.queue[os.start])
			os.start = (os.start + 1) % os.capacity
		}
		os.end = next
		os.queue[os.end] = item
		os.set[item] = struct{}{}
	}
}

// Has checks if certain items exists in the set
func (os *OrderedSet) Has(item interface{}) bool {
	os.lock.RLock()
	defer os.lock.RUnlock()

	_, ok := os.set[item]
	return ok
}

// Size returns the size of the set
func (os *OrderedSet) Size() int {
	os.lock.RLock()
	defer os.lock.RUnlock()

	return len(os.set)
}

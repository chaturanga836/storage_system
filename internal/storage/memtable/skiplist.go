package memtable

import (
	"math/rand"
	"sync"
)

// SkipList implements a concurrent skip list data structure
type SkipList struct {
	mu       sync.RWMutex
	header   *Node
	level    int
	maxLevel int
	length   int
}

// Node represents a node in the skip list
type Node struct {
	key     string
	value   interface{}
	forward []*Node
	mu      sync.RWMutex
}

// SkipListIterator provides iteration over skip list
type SkipListIterator struct {
	current *Node
	skiplist *SkipList
}

const (
	DefaultMaxLevel = 16
	P               = 0.5
)

// NewSkipList creates a new skip list
func NewSkipList(maxLevel int) *SkipList {
	if maxLevel <= 0 {
		maxLevel = DefaultMaxLevel
	}

	header := &Node{
		forward: make([]*Node, maxLevel),
	}

	return &SkipList{
		header:   header,
		maxLevel: maxLevel,
		level:    0,
	}
}

// Put inserts or updates a key-value pair
func (sl *SkipList) Put(key string, value interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	update := make([]*Node, sl.maxLevel)
	current := sl.header

	// Search for the position to insert
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
		update[i] = current
	}

	current = current.forward[0]

	// If key already exists, update value
	if current != nil && current.key == key {
		current.mu.Lock()
		current.value = value
		current.mu.Unlock()
		return
	}

	// Generate random level for new node
	newLevel := sl.randomLevel()

	// If new level is greater than current level, update level
	if newLevel > sl.level {
		for i := sl.level + 1; i <= newLevel; i++ {
			update[i] = sl.header
		}
		sl.level = newLevel
	}

	// Create new node
	newNode := &Node{
		key:     key,
		value:   value,
		forward: make([]*Node, newLevel+1),
	}

	// Update forward pointers
	for i := 0; i <= newLevel; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	sl.length++
}

// Get retrieves a value by key
func (sl *SkipList) Get(key string) interface{} {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	current := sl.header

	// Search for the key
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
	}

	current = current.forward[0]

	if current != nil && current.key == key {
		current.mu.RLock()
		value := current.value
		current.mu.RUnlock()
		return value
	}

	return nil
}

// Delete removes a key-value pair
func (sl *SkipList) Delete(key string) bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	update := make([]*Node, sl.maxLevel)
	current := sl.header

	// Search for the position to delete
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < key {
			current = current.forward[i]
		}
		update[i] = current
	}

	current = current.forward[0]

	// If key exists, delete it
	if current != nil && current.key == key {
		for i := 0; i <= sl.level; i++ {
			if update[i].forward[i] != current {
				break
			}
			update[i].forward[i] = current.forward[i]
		}

		// Update level if necessary
		for sl.level > 0 && sl.header.forward[sl.level] == nil {
			sl.level--
		}

		sl.length--
		return true
	}

	return false
}

// Contains checks if a key exists
func (sl *SkipList) Contains(key string) bool {
	return sl.Get(key) != nil
}

// Len returns the number of elements in the skip list
func (sl *SkipList) Len() int {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.length
}

// IsEmpty returns true if the skip list is empty
func (sl *SkipList) IsEmpty() bool {
	return sl.Len() == 0
}

// Range iterates over all key-value pairs with keys starting with the given prefix
func (sl *SkipList) Range(prefix string, fn func(key string, value interface{}) bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	current := sl.header.forward[0]

	for current != nil {
		if len(current.key) >= len(prefix) && current.key[:len(prefix)] == prefix {
			current.mu.RLock()
			shouldContinue := fn(current.key, current.value)
			current.mu.RUnlock()
			
			if !shouldContinue {
				break
			}
		}
		current = current.forward[0]
	}
}

// RangeFrom iterates over all key-value pairs starting from the given key
func (sl *SkipList) RangeFrom(startKey string, fn func(key string, value interface{}) bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	current := sl.header

	// Find the starting position
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].key < startKey {
			current = current.forward[i]
		}
	}

	current = current.forward[0]

	// Iterate from the starting position
	for current != nil {
		current.mu.RLock()
		shouldContinue := fn(current.key, current.value)
		current.mu.RUnlock()
		
		if !shouldContinue {
			break
		}
		
		current = current.forward[0]
	}
}

// Iterator returns an iterator for the skip list
func (sl *SkipList) Iterator() *SkipListIterator {
	return &SkipListIterator{
		current:  sl.header,
		skiplist: sl,
	}
}

// First returns the first key-value pair
func (sl *SkipList) First() (string, interface{}) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	if sl.header.forward[0] != nil {
		node := sl.header.forward[0]
		node.mu.RLock()
		key, value := node.key, node.value
		node.mu.RUnlock()
		return key, value
	}

	return "", nil
}

// Last returns the last key-value pair
func (sl *SkipList) Last() (string, interface{}) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	current := sl.header
	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil {
			current = current.forward[i]
		}
	}

	if current != sl.header {
		current.mu.RLock()
		key, value := current.key, current.value
		current.mu.RUnlock()
		return key, value
	}

	return "", nil
}

// Clear removes all elements from the skip list
func (sl *SkipList) Clear() {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.header.forward = make([]*Node, sl.maxLevel)
	sl.level = 0
	sl.length = 0
}

// randomLevel generates a random level for a new node
func (sl *SkipList) randomLevel() int {
	level := 0
	for rand.Float64() < P && level < sl.maxLevel-1 {
		level++
	}
	return level
}

// SkipListIterator methods

// Next advances the iterator to the next element
func (iter *SkipListIterator) Next() bool {
	iter.skiplist.mu.RLock()
	defer iter.skiplist.mu.RUnlock()

	if iter.current == nil {
		return false
	}

	iter.current = iter.current.forward[0]
	return iter.current != nil
}

// HasNext returns true if there are more elements
func (iter *SkipListIterator) HasNext() bool {
	iter.skiplist.mu.RLock()
	defer iter.skiplist.mu.RUnlock()

	return iter.current != nil && iter.current.forward[0] != nil
}

// Key returns the current key
func (iter *SkipListIterator) Key() string {
	if iter.current == nil {
		return ""
	}

	iter.current.mu.RLock()
	defer iter.current.mu.RUnlock()
	return iter.current.key
}

// Value returns the current value
func (iter *SkipListIterator) Value() interface{} {
	if iter.current == nil {
		return nil
	}

	iter.current.mu.RLock()
	defer iter.current.mu.RUnlock()
	return iter.current.value
}

// Reset resets the iterator to the beginning
func (iter *SkipListIterator) Reset() {
	iter.current = iter.skiplist.header
}

package maps

import "sync"

// Map is like a Go map[KeyType]ElementType but is safe for concurrent use
// by multiple goroutines.
//
// The zero Map is empty and ready for use. A Map must not be copied after first use.
//
// In the terminology of the Go memory model, Map arranges that a write operation
// “synchronizes before” any read operation that observes the effect of the write, where
// read and write operations are defined as follows.
// Load, LoadAndDelete, LoadOrStore, Swap, CompareAndSwap, and CompareAndDelete
// are read operations; Delete, LoadAndDelete, Store, and Swap are write operations;
// LoadOrStore is a write operation when it returns loaded set to false;
// CompareAndSwap is a write operation when it returns swapped set to true;
// and CompareAndDelete is a write operation when it returns deleted set to true.
type Map[K comparable, V any] struct {
	mu    sync.Mutex
	dirty map[K]V
}

// Load returns the value stored in the map for a key, or nil if no
// value is present.
// The ok result indicates whether value was found in the map.
func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	m.mu.Lock()
	value, ok = m.dirty[key]
	m.mu.Unlock()
	return
}

// Store sets the value for a key.
func (m *Map[K, V]) Store(key K, value V) {
	m.mu.Lock()
	if m.dirty == nil {
		m.dirty = make(map[K]V)
	}
	m.dirty[key] = value
	m.mu.Unlock()
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	m.mu.Lock()
	actual, loaded = m.dirty[key]
	if !loaded {
		actual = value
		if m.dirty == nil {
			m.dirty = make(map[K]V)
		}
		m.dirty[key] = value
	}
	m.mu.Unlock()
	return
}

// Swap swaps the value for a key and returns the previous value if any.
// The loaded result reports whether the key was present.
func (m *Map[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	m.mu.Lock()
	if m.dirty == nil {
		m.dirty = make(map[K]V)
	}

	previous, loaded = m.dirty[key]
	m.dirty[key] = value
	m.mu.Unlock()
	return
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	m.mu.Lock()
	value, loaded = m.dirty[key]
	if loaded {
		delete(m.dirty, key)
	}
	m.mu.Unlock()
	return
}

// Delete deletes the value for a key.
func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	delete(m.dirty, key)
	m.mu.Unlock()
}

// CompareAndSwap swaps the old and new values for key
// if the value stored in the map is equal to old.
// The old value must be of a comparable type.
func (m *Map[K, V]) CompareAndSwap(key K, old, new V) (swapped bool) {
	m.mu.Lock()
	value, loaded := m.dirty[key]
	if loaded && any(value) == any(old) {
		m.dirty[key] = new
		swapped = true
	}
	m.mu.Unlock()
	return
}

// CompareAndDelete deletes the entry for key if its value is equal to old.
// The old value must be of a comparable type.
//
// If there is no current value for key in the map, CompareAndDelete
// returns false.
func (m *Map[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	m.mu.Lock()
	value, loaded := m.dirty[key]
	if loaded && any(value) == any(old) {
		delete(m.dirty, key)
		deleted = true
	}
	m.mu.Unlock()
	return
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently (including by f), Range may reflect any
// mapping for that key from any point during the Range call. Range does not
// block other methods on the receiver; even f itself may call any method on m.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (m *Map[K, V]) Range(f func(key K, value V) (shouldContinue bool)) {
	m.mu.Lock()
	keys := make([]K, 0, len(m.dirty))
	for k := range m.dirty {
		keys = append(keys, k)
	}
	m.mu.Unlock()

	for _, k := range keys {
		v, ok := m.Load(k)
		if !ok {
			continue
		}
		if !f(k, v) {
			break
		}
	}
}

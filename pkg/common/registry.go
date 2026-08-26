package common

import "sync"

// Registry is a generic, thread-safe concurrent key-value store.
type Registry[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
}

// NewRegistry creates a new generic Registry instance.
func NewRegistry[K comparable, V any]() *Registry[K, V] {
	return &Registry[K, V]{
		items: make(map[K]V),
	}
}

// Set stores a key-value pair.
func (r *Registry[K, V]) Set(key K, val V) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[key] = val
}

// Get retrieves a value by key safely.
func (r *Registry[K, V]) Get(key K) (V, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.items[key]
	return val, ok
}

// Delete removes a key from the registry.
func (r *Registry[K, V]) Delete(key K) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, key)
}

// Len returns the current number of elements.
func (r *Registry[K, V]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.items)
}

// Range iterates over items with a read lock.
func (r *Registry[K, V]) Range(fn func(key K, val V) bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for k, v := range r.items {
		if !fn(k, v) {
			break
		}
	}
}

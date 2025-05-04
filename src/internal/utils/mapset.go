package utils


import (
	"sync"
  	mapset "github.com/deckarep/golang-set/v2"
)


type MapSet[K comparable, V comparable] struct {
	ms map[K]mapset.Set[V]
	lock sync.RWMutex
}


func (ms *MapSet[K, V]) store (key K, value V) {
	curval, ok := ms.ms[key]
	if !ok {
		curval = mapset.NewThreadUnsafeSet[V]()
	}
	curval.Add(value)
	ms.ms[key] = curval
}

func (ms *MapSet[K, V]) Store (key K, value V) {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	ms.store(key, value)
}

func (ms *MapSet[K, V]) StoreMany (key K, values []V) {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	for _, value := range values {
		ms.store(key, value)
	}
}


func (ms *MapSet[K, V]) Load (key K) (mapset.Set[V], bool) {
	ms.lock.RLock()
	defer ms.lock.Unlock()
	val, ok := ms.ms[key]
	return val, ok
}

func (ms *MapSet[K, V]) Delete (key K) {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	delete(ms.ms, key)
}

func (ms *MapSet[K, V]) DeleteVal (key K, value V) {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	curval, ok := ms.ms[key]
	if !ok {
		return
	}
	curval.Remove(value)
	if curval.IsEmpty() {
		delete(ms.ms, key)
	}
}

func (ms *MapSet[K, V]) ContainsVal (key K, value V) bool {
	ms.lock.RLock()
	defer ms.lock.Unlock()
	curval, ok := ms.ms[key]
	if !ok {
		return false
	}
	return curval.Contains(value)
}

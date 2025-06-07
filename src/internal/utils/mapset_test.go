package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapSetStoreMultipleValues(t *testing.T) {
	ms := NewMapSet[string, int]()

	// Store multiple values for the same key
	ms.Store("key1", 1)
	ms.Store("key1", 2)
	ms.Store("key1", 3)

	set, ok := ms.Load("key1")
	assert.True(t, ok, "Expected key1 to exist")
	assert.Equal(t, 3, set.Cardinality(), "Expected set cardinality 3")
	assert.True(t, set.Contains(1), "Expected set to contain value 1")
	assert.True(t, set.Contains(2), "Expected set to contain value 2")
	assert.True(t, set.Contains(3), "Expected set to contain value 3")
}

func TestMapSetStoreMany(t *testing.T) {
	ms := NewMapSet[string, int]()

	values := []int{10, 20, 30, 40}
	ms.StoreMany("key1", values)

	set, ok := ms.Load("key1")
	assert.True(t, ok, "Expected key1 to exist")
	assert.Equal(t, 4, set.Cardinality(), "Expected set cardinality 4")
	for _, v := range values {
		assert.True(t, set.Contains(v), "Expected set to contain value %d", v)
	}
}

func TestMapSetStoreManyEmpty(t *testing.T) {
	ms := NewMapSet[string, int]()

	ms.StoreMany("key1", []int{})

	_, ok := ms.Load("key1")
	assert.False(t, ok, "Expected key1 to not exist after storing empty slice")
}

func TestMapSetLoad(t *testing.T) {
	ms := NewMapSet[string, int]()

	// Test loading non-existent key
	_, ok := ms.Load("nonexistent")
	assert.False(t, ok, "Expected non-existent key to return false")

	// Test loading existing key
	ms.Store("key1", 42)
	set, ok := ms.Load("key1")
	assert.True(t, ok, "Expected key1 to exist")
	assert.True(t, set.Contains(42), "Expected set to contain value 42")
}

func TestMapSetDelete(t *testing.T) {
	ms := NewMapSet[string, int]()

	// Store some values
	ms.Store("key1", 1)
	ms.Store("key1", 2)
	ms.Store("key2", 3)

	// Delete key1
	ms.Delete("key1")

	_, ok := ms.Load("key1")
	assert.False(t, ok, "Expected key1 to be deleted")

	// key2 should still exist
	set, ok := ms.Load("key2")
	assert.True(t, ok, "Expected key2 to still exist")
	assert.True(t, set.Contains(3), "Expected key2 to still contain value 3")
}

func TestMapSetDeleteVal(t *testing.T) {
	ms := NewMapSet[string, int]()

	// Store multiple values for a key
	ms.Store("key1", 1)
	ms.Store("key1", 2)
	ms.Store("key1", 3)

	// Delete one value
	ms.DeleteVal("key1", 2)

	set, ok := ms.Load("key1")
	assert.True(t, ok, "Expected key1 to still exist")
	assert.False(t, set.Contains(2), "Expected value 2 to be deleted")
	assert.True(t, set.Contains(1), "Expected value 1 to still exist")
	assert.True(t, set.Contains(3), "Expected value 3 to still exist")
	assert.Equal(t, 2, set.Cardinality(), "Expected set cardinality 2")
}

func TestMapSetDeleteValLastValue(t *testing.T) {
	ms := NewMapSet[string, int]()

	// Store single value
	ms.Store("key1", 1)

	// Delete the only value
	ms.DeleteVal("key1", 1)

	// Key should be completely removed
	_, ok := ms.Load("key1")
	assert.False(t, ok, "Expected key1 to be completely removed after deleting last value")
}

func TestMapSetDeleteValNonExistentValue(t *testing.T) {
	ms := NewMapSet[string, int]()

	ms.Store("key1", 1)
	ms.Store("key1", 2)

	// Delete non-existent value
	ms.DeleteVal("key1", 99)

	// Should not affect existing values
	set, ok := ms.Load("key1")
	assert.True(t, ok, "Expected key1 to still exist")
	assert.Equal(t, 2, set.Cardinality(), "Expected set cardinality 2")
	assert.True(t, set.Contains(1), "Expected set to contain value 1")
	assert.True(t, set.Contains(2), "Expected set to contain value 2")
}

func TestMapSetContainsVal(t *testing.T) {
	ms := NewMapSet[string, int]()

	// Test non-existent key
	assert.False(t, ms.ContainsVal("nonexistent", 1), "Expected ContainsVal to return false for non-existent key")

	// Store some values
	ms.Store("key1", 1)
	ms.Store("key1", 2)

	// Test existing values
	assert.True(t, ms.ContainsVal("key1", 1), "Expected ContainsVal to return true for existing value 1")
	assert.True(t, ms.ContainsVal("key1", 2), "Expected ContainsVal to return true for existing value 2")

	// Test non-existent value
	assert.False(t, ms.ContainsVal("key1", 99), "Expected ContainsVal to return false for non-existent value")
}

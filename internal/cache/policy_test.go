package cache

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager(nil, 500, PolicyLRU)

	if mgr.policy != PolicyLRU {
		t.Errorf("Expected policy LRU, got %s", mgr.policy)
	}

	expectedBytes := int64(500 * 1024 * 1024)
	if mgr.maxSize != expectedBytes {
		t.Errorf("Expected size %d, got %d", expectedBytes, mgr.maxSize)
	}
}

func TestPolicyTypes(t *testing.T) {
	if PolicyLRU != "LRU" {
		t.Error("LRU constant mismatch")
	}
	if PolicyFIFO != "FIFO" {
		t.Error("FIFO constant mismatch")
	}
}

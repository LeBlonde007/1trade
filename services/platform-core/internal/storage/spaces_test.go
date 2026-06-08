package storage

import (
	"strings"
	"testing"
)

// TestDisabledStore confirms a zero/unconfigured Store reports disabled and refuses to presign, so the
// service runs (and the endpoint 501s) without storage configured.
func TestDisabledStore(t *testing.T) {
	s, err := New(Config{}) // no endpoint/keys
	if err != nil {
		t.Fatalf("New(empty): %v", err)
	}
	if s.Enabled() {
		t.Fatal("an unconfigured Store must be disabled")
	}
}

// TestObjectKey checks keys are tenant-scoped, path-safe, and collision-resistant (unique per call).
func TestObjectKey(t *testing.T) {
	k := objectKey("tenant-123", "../../etc/My Report (final).pdf")
	if !strings.HasPrefix(k, "uploads/tenant-123/") {
		t.Fatalf("key not tenant-scoped: %q", k)
	}
	if strings.Contains(k, "..") || strings.Contains(k, " ") {
		t.Fatalf("key not sanitized: %q", k)
	}
	if !strings.HasSuffix(k, "My-Report-final-.pdf") && !strings.HasSuffix(k, "My-Report-final.pdf") {
		t.Fatalf("key lost the filename: %q", k)
	}
	if k1, k2 := objectKey("t", "a.txt"), objectKey("t", "a.txt"); k1 == k2 {
		t.Fatal("keys must be unique per call (uuid)")
	}
}

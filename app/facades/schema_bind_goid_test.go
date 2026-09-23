package facades

import (
	"sync"
	"testing"
	"time"
)

func TestGoroutineIDDistinct(t *testing.T) {
	t.Parallel()
	var a, b int64
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		a = goroutineID()
		time.Sleep(20 * time.Millisecond)
	}()
	go func() {
		defer wg.Done()
		b = goroutineID()
		time.Sleep(20 * time.Millisecond)
	}()
	wg.Wait()
	if a == 0 || b == 0 {
		t.Fatalf("goid zero a=%d b=%d", a, b)
	}
	if a == b {
		t.Fatalf("expected distinct goids, both %d", a)
	}
}

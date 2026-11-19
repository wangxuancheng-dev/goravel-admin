package feedback

import (
	"context"
	"testing"
)

func TestCreateRequiresContent(t *testing.T) {
	ResolveQuery = nil
	ResolveActorID = nil
	_, err := NewService().Create(context.Background(), "alice", "   ")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

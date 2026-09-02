package store

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/trade1/platform-core/internal/domain"
	"github.com/google/uuid"
)

// TestConversationsIntegration exercises conversation CRUD plus tenant isolation against real Postgres
// (one tenant must never see, read, update, or delete another's). Skips unless DATABASE_URL is set.
func TestConversationsIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping conversations integration test")
	}
	ctx := context.Background()
	s, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer s.Close()

	hash, _ := domain.HashPassword("pw")
	a, err := s.Signup(ctx, "a+"+uuid.NewString()+"@x.ai", hash, "A")
	if err != nil {
		t.Fatalf("signup A: %v", err)
	}
	b, err := s.Signup(ctx, "b+"+uuid.NewString()+"@x.ai", hash, "B")
	if err != nil {
		t.Fatalf("signup B: %v", err)
	}

	c, err := s.CreateConversation(ctx, a.TenantID, a.ID, "First chat", "llama-3.1-8b", json.RawMessage(`[{"role":"user","text":"hi"}]`))
	if err != nil || c.ID == "" || c.Title != "First chat" {
		t.Fatalf("create: %+v err=%v", c, err)
	}

	// A can list + read its own conversation.
	if list, err := s.ListConversations(ctx, a.TenantID, 50); err != nil || len(list) != 1 || list[0].ID != c.ID {
		t.Fatalf("list A: %v %+v", err, list)
	}
	if got, found, err := s.GetConversation(ctx, a.TenantID, c.ID); err != nil || !found || len(got.Transcript) == 0 {
		t.Fatalf("get A: ok=%v err=%v", found, err)
	}

	// Tenant isolation: B must not see / read / update / delete A's conversation.
	if bl, _ := s.ListConversations(ctx, b.TenantID, 50); len(bl) != 0 {
		t.Fatalf("B saw A's conversations: %+v", bl)
	}
	if _, found, _ := s.GetConversation(ctx, b.TenantID, c.ID); found {
		t.Fatal("B read A's conversation")
	}
	if _, found, _ := s.UpdateConversation(ctx, b.TenantID, c.ID, "", "", json.RawMessage(`[]`)); found {
		t.Fatal("B updated A's conversation")
	}
	if ok, _ := s.DeleteConversation(ctx, b.TenantID, c.ID); ok {
		t.Fatal("B deleted A's conversation")
	}

	// A can update + delete its own.
	if upd, found, err := s.UpdateConversation(ctx, a.TenantID, c.ID, "Renamed", "", json.RawMessage(`[{"role":"user","text":"hi"}]`)); err != nil || !found || upd.Title != "Renamed" {
		t.Fatalf("update A: ok=%v err=%v title=%q", found, err, upd.Title)
	}
	if ok, err := s.DeleteConversation(ctx, a.TenantID, c.ID); err != nil || !ok {
		t.Fatalf("delete A: ok=%v err=%v", ok, err)
	}
	if _, found, _ := s.GetConversation(ctx, a.TenantID, c.ID); found {
		t.Fatal("conversation still present after delete")
	}
}

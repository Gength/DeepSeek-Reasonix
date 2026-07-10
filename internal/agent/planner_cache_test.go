package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"reasonix/internal/provider"
)

func TestSaveAndLoadPlannerCache(t *testing.T) {
	cacheDir := t.TempDir()
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "system prompt"},
		{Role: provider.RoleUser, Content: "first user input"},
		{Role: provider.RoleAssistant, Content: "first plan"},
	}

	// Save the cache.
	if err := SavePlannerCache(msgs, cacheDir); err != nil {
		t.Fatalf("SavePlannerCache: %v", err)
	}

	// Load it back.
	loaded := LoadPlannerCache(cacheDir)
	if len(loaded) != len(msgs) {
		t.Fatalf("got %d messages, want %d", len(loaded), len(msgs))
	}
	for i, m := range msgs {
		if loaded[i].Role != m.Role || loaded[i].Content != m.Content {
			t.Errorf("message %d: got %+v, want %+v", i, loaded[i], m)
		}
	}
}

func TestLoadPlannerCacheMissing(t *testing.T) {
	cacheDir := t.TempDir()
	loaded := LoadPlannerCache(cacheDir)
	if loaded != nil {
		t.Errorf("expected nil for missing cache, got %d messages", len(loaded))
	}
}

func TestLoadPlannerCacheCorrupt(t *testing.T) {
	cacheDir := t.TempDir()
	dir := filepath.Join(cacheDir, "planner")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "planner-cache.json"), []byte("not valid json"), 0o644)

	loaded := LoadPlannerCache(cacheDir)
	if loaded != nil {
		t.Errorf("expected nil for corrupt cache, got %d messages", len(loaded))
	}
}

func TestLoadPlannerCacheExpired(t *testing.T) {
	cacheDir := t.TempDir()
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "hello"},
	}

	// Save then manually override UpdatedAt to be >TTL in the past.
	if err := SavePlannerCache(msgs, cacheDir); err != nil {
		t.Fatalf("SavePlannerCache: %v", err)
	}
	// Corrupt the file to set an old timestamp.
	old := plannerCache{
		Version:   plannerCacheVersion,
		CreatedAt: time.Now().Add(-2 * PlannerCacheTTL),
		UpdatedAt: time.Now().Add(-2 * PlannerCacheTTL),
		Messages:  msgs,
	}
	writePlannerCache(plannerCachePath(cacheDir), &old)

	loaded := LoadPlannerCache(cacheDir)
	if loaded != nil {
		t.Errorf("expected nil for expired cache, got %d messages", len(loaded))
	}

	// Verify the file was cleaned up.
	if _, err := os.Stat(plannerCachePath(cacheDir)); !os.IsNotExist(err) {
		t.Errorf("expected expired cache file to be removed, stat error: %v", err)
	}
}

func TestSavePlannerCacheEmptyCacheDir(t *testing.T) {
	// Must not panic or error.
	if err := SavePlannerCache([]provider.Message{{Role: provider.RoleUser, Content: "hi"}}, ""); err != nil {
		t.Errorf("empty cache dir should be a no-op, got: %v", err)
	}
}

func TestSavePlannerCacheEmptyMessages(t *testing.T) {
	cacheDir := t.TempDir()
	if err := SavePlannerCache(nil, cacheDir); err != nil {
		t.Errorf("nil messages should be a no-op, got: %v", err)
	}
	if err := SavePlannerCache([]provider.Message{}, cacheDir); err != nil {
		t.Errorf("empty messages should be a no-op, got: %v", err)
	}

	// Verify no file was created.
	if _, err := os.Stat(plannerCachePath(cacheDir)); !os.IsNotExist(err) {
		t.Error("expected no cache file for empty messages")
	}
}

func TestPlannerCacheUpdatedAtRefreshedOnUpdate(t *testing.T) {
	cacheDir := t.TempDir()
	first := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "first turn"},
	}

	// First save.
	if err := SavePlannerCache(first, cacheDir); err != nil {
		t.Fatalf("first save: %v", err)
	}
	raw := loadPlannerCacheRaw(plannerCachePath(cacheDir))
	if raw == nil || raw.CreatedAt.IsZero() {
		t.Fatal("createdAt should be set after first save")
	}
	firstCreatedAt := raw.CreatedAt
	firstUpdatedAt := raw.UpdatedAt
	if firstUpdatedAt.IsZero() {
		t.Fatal("updatedAt should be set after first save")
	}

	// Sleep briefly so a new timestamp would differ.
	time.Sleep(10 * time.Millisecond)

	// Second save with more messages — createdAt must stay the same,
	// updatedAt must be refreshed.
	second := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "first turn"},
		{Role: provider.RoleAssistant, Content: "first plan"},
		{Role: provider.RoleUser, Content: "second turn"},
	}
	if err := SavePlannerCache(second, cacheDir); err != nil {
		t.Fatalf("second save: %v", err)
	}
	raw = loadPlannerCacheRaw(plannerCachePath(cacheDir))
	if raw == nil {
		t.Fatal("cache should exist after second save")
	}
	if !raw.CreatedAt.Equal(firstCreatedAt) {
		t.Errorf("createdAt changed: was %v, now %v", firstCreatedAt, raw.CreatedAt)
	}
	if raw.UpdatedAt.Equal(firstUpdatedAt) {
		t.Errorf("updatedAt should have been refreshed but stayed: %v", raw.UpdatedAt)
	}
	if len(raw.Messages) != len(second) {
		t.Errorf("messages length: got %d, want %d", len(raw.Messages), len(second))
	}
}

func TestClearPlannerCache(t *testing.T) {
	cacheDir := t.TempDir()
	msgs := []provider.Message{{Role: provider.RoleUser, Content: "hello"}}
	if err := SavePlannerCache(msgs, cacheDir); err != nil {
		t.Fatalf("SavePlannerCache: %v", err)
	}
	if _, err := os.Stat(plannerCachePath(cacheDir)); os.IsNotExist(err) {
		t.Fatal("cache file should exist before Clear")
	}

	ClearPlannerCache(cacheDir)
	if _, err := os.Stat(plannerCachePath(cacheDir)); !os.IsNotExist(err) {
		t.Error("cache file should be removed after Clear")
	}
}

func TestClearPlannerCacheEmptyCacheDir(t *testing.T) {
	// Must not panic.
	ClearPlannerCache("")
}

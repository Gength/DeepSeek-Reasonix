package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"reasonix/internal/provider"
)

// plannerCache is the on-disk format for the planner's temporary session cache.
// It keeps the conversation history across agent restarts so the planner does
// not lose context, but automatically expires after PlannerCacheTTL from the
// last save (UpdatedAt).
type plannerCache struct {
	Version   int                `json:"version"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Messages  []provider.Message `json:"messages"`
}

// PlannerCacheTTL is the time-to-live for planner session cache files.
// Caches older than this are discarded on load. Exported as a var so
// it can be adjusted at runtime (e.g., for testing or configuration).
var PlannerCacheTTL = 4 * time.Hour

const (
	plannerCacheVersion  = 1
	plannerCacheFileName = "planner-cache.json"
	plannerCacheSubDir   = "planner"
)

// plannerCachePath returns the full path to the planner cache file under cacheDir.
func plannerCachePath(cacheDir string) string {
	return filepath.Join(cacheDir, plannerCacheSubDir, plannerCacheFileName)
}

// SavePlannerCache persists the planner's session messages to a temporary file.
// CreatedAt is set once on the first save; UpdatedAt is refreshed on every save
// so the TTL always counts from the last update.
func SavePlannerCache(messages []provider.Message, cacheDir string) error {
	if cacheDir == "" || len(messages) == 0 {
		return nil
	}
	path := plannerCachePath(cacheDir)

	// Load existing cache to preserve the original createdAt.
	existing := loadPlannerCacheRaw(path)
	if existing == nil {
		existing = &plannerCache{
			Version:   plannerCacheVersion,
			CreatedAt: time.Now(),
		}
	}
	existing.UpdatedAt = time.Now()
	existing.Version = plannerCacheVersion
	existing.Messages = messages
	return writePlannerCache(path, existing)
}

func writePlannerCache(path string, c *plannerCache) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// loadPlannerCacheRaw reads the cache file without TTL check. Returns nil on any
// error (missing, corrupt, unreadable) — caching is best-effort.
func loadPlannerCacheRaw(path string) *plannerCache {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var c plannerCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	return &c
}

// LoadPlannerCache reads a previously saved planner cache. Returns the messages
// when the cache exists and has not expired (TTL < PlannerCacheTTL from
// UpdatedAt, with fallback to CreatedAt for backward compatibility).
// Returns nil when the cache is missing, expired, corrupt, or caching is
// unavailable — the caller falls back to a fresh session.
func LoadPlannerCache(cacheDir string) []provider.Message {
	if cacheDir == "" {
		return nil
	}
	path := plannerCachePath(cacheDir)
	c := loadPlannerCacheRaw(path)
	if c == nil || len(c.Messages) == 0 {
		return nil
	}
	last := c.UpdatedAt
	if last.IsZero() {
		last = c.CreatedAt
	}
	if last.IsZero() || time.Since(last) > PlannerCacheTTL {
		// Expired — clean up the stale file.
		os.Remove(path)
		// Also remove the parent dir if empty.
		os.Remove(filepath.Dir(path))
		return nil
	}
	return c.Messages
}

// ClearPlannerCache removes the planner cache file entirely. Used during session
// reset to prevent stale context from leaking across branches or tabs.
func ClearPlannerCache(cacheDir string) {
	if cacheDir == "" {
		return
	}
	path := plannerCachePath(cacheDir)
	os.Remove(path)
	os.Remove(filepath.Dir(path))
}

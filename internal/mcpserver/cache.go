package mcpserver

import (
	"container/list"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

const pipelineCacheTTL = 5 * time.Minute

// pipelineCache is a session-scoped LRU with a 5-minute TTL for
// fibe_pipeline outputs. The cache is indexed by session ID + pipeline ID so
// a session cannot read another tenant's pipeline results even with a valid
// pipeline_id.
type pipelineCache struct {
	mu          sync.Mutex
	maxEntries  int
	maxEntryLen int // bytes; 0 = unlimited
	entries     map[cacheKey]*list.Element
	order       *list.List
}

type cacheKey struct {
	sessionID  string
	pipelineID string
}

type cacheEntry struct {
	key       cacheKey
	payload   json.RawMessage
	truncated bool
	storedAt  time.Time
}

func newPipelineCache(maxEntries, maxEntryLen int) *pipelineCache {
	if maxEntries <= 0 {
		maxEntries = 0
	}
	return &pipelineCache{
		maxEntries:  maxEntries,
		maxEntryLen: maxEntryLen,
		entries:     make(map[cacheKey]*list.Element),
		order:       list.New(),
	}
}

// Put stores value under a newly-minted pipeline ID and returns it. Returns
// an empty ID if caching is disabled (maxEntries == 0).
func (c *pipelineCache) Put(sessionID string, value any) (pipelineID string, truncated bool, err error) {
	if c == nil || c.maxEntries == 0 {
		return "", false, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return "", false, err
	}

	if c.maxEntryLen > 0 && len(payload) > c.maxEntryLen {
		payload = []byte(`{"truncated":true,"reason":"result exceeded FIBE_MCP_PIPELINE_CACHE_ENTRY_MAX"}`)
		truncated = true
	}

	id := uuid.NewString()
	key := cacheKey{sessionID: sessionID, pipelineID: id}
	entry := &cacheEntry{
		key:       key,
		payload:   payload,
		truncated: truncated,
		storedAt:  time.Now(),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	elem := c.order.PushFront(entry)
	c.entries[key] = elem

	// Evict LRU if we're over capacity.
	for c.order.Len() > c.maxEntries {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}
		c.order.Remove(oldest)
		delete(c.entries, oldest.Value.(*cacheEntry).key)
	}

	return id, truncated, nil
}

// Get returns the cached payload (as raw JSON) or nil if not found/expired.
// Expired entries are removed on lookup.
func (c *pipelineCache) Get(sessionID, pipelineID string) (json.RawMessage, bool) {
	if c == nil || c.maxEntries == 0 {
		return nil, false
	}
	key := cacheKey{sessionID: sessionID, pipelineID: pipelineID}

	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	entry := elem.Value.(*cacheEntry)
	if time.Since(entry.storedAt) > pipelineCacheTTL {
		c.order.Remove(elem)
		delete(c.entries, key)
		return nil, false
	}
	c.order.MoveToFront(elem)
	return entry.payload, true
}

// Stats returns a snapshot of cache health. Kept internal for now; may be
// surfaced via an admin resource in a later phase.
func (c *pipelineCache) Stats() (entries, capacity int) {
	if c == nil {
		return 0, 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len(), c.maxEntries
}

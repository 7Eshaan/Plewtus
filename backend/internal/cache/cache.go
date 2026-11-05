package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"plewtus/internal/model"
)

// entry holds an item in the LRU cache with its expiration.
type entry struct {
	key       string
	response  *model.SearchResponse
	expiresAt time.Time
}

// Options configures the in-memory cache.
type Options struct {
	TTL        time.Duration // Time-to-live for cached entries (0 disables caching)
	MaxEntries int           // Maximum number of entries before LRU eviction (default 1000)
}

// SearchCache defines the interface for caching search responses.
type SearchCache interface {
	Get(key string) (*model.SearchResponse, bool)
	Set(key string, resp *model.SearchResponse)
	Len() int
	Clear()
	Key(req *model.SearchRequest) string
}

// MemoryCache implements an in-memory, thread-safe LRU cache with TTL expiration.
type MemoryCache struct {
	mu         sync.RWMutex
	ttl        time.Duration
	maxEntries int
	evictList  *list.List
	items      map[string]*list.Element
}

// New creates a new in-memory LRU search cache.
func New(opts Options) *MemoryCache {
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = 1000
	}
	return &MemoryCache{
		ttl:        opts.TTL,
		maxEntries: opts.MaxEntries,
		evictList:  list.New(),
		items:      make(map[string]*list.Element),
	}
}

// Key computes a deterministic, canonical cache key for a SearchRequest.
func (c *MemoryCache) Key(req *model.SearchRequest) string {
	if req == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("q:")
	sb.WriteString(strings.ToLower(strings.TrimSpace(req.Query)))
	sb.WriteString("|l:")
	sb.WriteString(fmt.Sprintf("%d", req.Limit))
	sb.WriteString("|o:")
	sb.WriteString(fmt.Sprintf("%d", req.Offset))

	// Providers (sorted)
	var provs []string
	if len(req.Providers) > 0 {
		provs = append([]string(nil), req.Providers...)
		sort.Strings(provs)
	}
	sb.WriteString("|p:")
	sb.WriteString(strings.Join(provs, ","))

	// Filters
	sb.WriteString("|f:")
	sb.WriteString(strings.ToLower(strings.TrimSpace(req.Freshness)))
	sb.WriteString("|c:")
	sb.WriteString(strings.ToLower(strings.TrimSpace(req.Country)))
	sb.WriteString("|lang:")
	sb.WriteString(strings.ToLower(strings.TrimSpace(req.Language)))

	// Include domains (sorted)
	var inc []string
	if len(req.IncludeDomains) > 0 {
		inc = append([]string(nil), req.IncludeDomains...)
		sort.Strings(inc)
	}
	sb.WriteString("|inc:")
	sb.WriteString(strings.Join(inc, ","))

	// Exclude domains (sorted)
	var exc []string
	if len(req.ExcludeDomains) > 0 {
		exc = append([]string(nil), req.ExcludeDomains...)
		sort.Strings(exc)
	}
	sb.WriteString("|exc:")
	sb.WriteString(strings.Join(exc, ","))

	// Extract options
	if req.Extract != nil && req.Extract.Enabled {
		sb.WriteString("|ext:1,m:")
		sb.WriteString(strings.ToLower(strings.TrimSpace(req.Extract.Mode)))
		sb.WriteString(fmt.Sprintf(",p:%d,c:%d", req.Extract.MaxPages, req.Extract.MaxCharsPerPage))
	} else {
		sb.WriteString("|ext:0")
	}

	raw := sb.String()
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// Get retrieves a response from the cache if present and not expired.
func (c *MemoryCache) Get(key string) (*model.SearchResponse, bool) {
	if c.ttl <= 0 || key == "" {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	elem, found := c.items[key]
	if !found {
		return nil, false
	}

	ent := elem.Value.(*entry)

	// Check TTL expiration
	if time.Now().After(ent.expiresAt) {
		c.removeElement(elem)
		return nil, false
	}

	// Move to front (most recently used)
	c.evictList.MoveToFront(elem)

	// Return a clone marked as cached so mutations don't alter the cache
	cloned := cloneSearchResponse(ent.response)
	cloned.Metadata.Cached = true
	return cloned, true
}

// Set stores a response in the cache with the configured TTL.
func (c *MemoryCache) Set(key string, resp *model.SearchResponse) {
	if c.ttl <= 0 || key == "" || resp == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiresAt := now.Add(c.ttl)

	// If entry already exists, update it and move to front
	if elem, found := c.items[key]; found {
		c.evictList.MoveToFront(elem)
		ent := elem.Value.(*entry)
		ent.response = cloneSearchResponse(resp)
		ent.expiresAt = expiresAt
		return
	}

	// Evict oldest if capacity exceeded
	for c.evictList.Len() >= c.maxEntries {
		c.removeOldest()
	}

	// Add new entry
	ent := &entry{
		key:       key,
		response:  cloneSearchResponse(resp),
		expiresAt: expiresAt,
	}
	elem := c.evictList.PushFront(ent)
	c.items[key] = elem
}

// Len returns the current number of cached entries.
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.evictList.Len()
}

// Clear flushes all entries from the cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictList.Init()
	c.items = make(map[string]*list.Element)
}

func (c *MemoryCache) removeElement(elem *list.Element) {
	c.evictList.Remove(elem)
	ent := elem.Value.(*entry)
	delete(c.items, ent.key)
}

func (c *MemoryCache) removeOldest() {
	elem := c.evictList.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// cloneSearchResponse creates a deep copy of a SearchResponse.
func cloneSearchResponse(src *model.SearchResponse) *model.SearchResponse {
	if src == nil {
		return nil
	}

	dst := &model.SearchResponse{
		Query:    src.Query,
		Results:  make([]model.SearchResult, len(src.Results)),
		Metadata: src.Metadata,
	}

	// Deep copy results slice and engine sources
	for i, r := range src.Results {
		dst.Results[i] = r
		if len(r.Engines) > 0 {
			dst.Results[i].Engines = make([]model.EngineSource, len(r.Engines))
			copy(dst.Results[i].Engines, r.Engines)
		}
		if r.Content != nil {
			cnt := *r.Content
			if len(r.Content.Highlights) > 0 {
				cnt.Highlights = make([]string, len(r.Content.Highlights))
				copy(cnt.Highlights, r.Content.Highlights)
			}
			dst.Results[i].Content = &cnt
		}
	}

	// Deep copy metadata slices
	dst.Metadata.Providers.Queried = append([]string(nil), src.Metadata.Providers.Queried...)
	dst.Metadata.Providers.Successful = append([]string(nil), src.Metadata.Providers.Successful...)
	dst.Metadata.Providers.Failed = append([]model.ProviderError(nil), src.Metadata.Providers.Failed...)
	if src.Metadata.Providers.TimingMs != nil {
		dst.Metadata.Providers.TimingMs = make(map[string]int64, len(src.Metadata.Providers.TimingMs))
		for name, tookMs := range src.Metadata.Providers.TimingMs {
			dst.Metadata.Providers.TimingMs[name] = tookMs
		}
	}

	if src.Metadata.Filters != nil {
		flt := *src.Metadata.Filters
		flt.IncludeDomains = append([]string(nil), src.Metadata.Filters.IncludeDomains...)
		flt.ExcludeDomains = append([]string(nil), src.Metadata.Filters.ExcludeDomains...)
		dst.Metadata.Filters = &flt
	}

	return dst
}

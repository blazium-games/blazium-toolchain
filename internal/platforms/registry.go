package platforms

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	byID     = map[string]Platform{}
	planned  = map[string]Info{}
)

// Register a live platform implementation.
func Register(p Platform) {
	mu.Lock()
	defer mu.Unlock()
	info := p.Info()
	byID[info.ID] = p
}

// RegisterPlanned reserves an id (ps2, ps3, …) so `list` shows it and commands fail cleanly.
func RegisterPlanned(info Info) {
	if info.Status == "" {
		info.Status = StatusPlanned
	}
	mu.Lock()
	defer mu.Unlock()
	planned[info.ID] = info
}

// Lookup returns a supported platform or an error that includes planned ids.
func Lookup(id string) (Platform, error) {
	mu.RLock()
	defer mu.RUnlock()
	if p, ok := byID[id]; ok {
		return p, nil
	}
	if info, ok := planned[id]; ok {
		return nil, fmt.Errorf("%w: %s (%s) is reserved for a future release", ErrPlanned, info.ID, info.Name)
	}
	return nil, fmt.Errorf("%w: %q is not a platform", ErrUnknownPlatform, id)
}

// List returns supported then planned, sorted by id.
func List() []Info {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Info, 0, len(byID)+len(planned))
	for _, p := range byID {
		out = append(out, p.Info())
	}
	for _, info := range planned {
		if _, live := byID[info.ID]; live {
			continue
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

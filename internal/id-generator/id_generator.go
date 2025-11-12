package idgenerator

import "sync"

const maxInt64 = 1<<63 - 1

// IDGenerator is a thread-safe generator for unique int64 IDs.
// It ensures that IDs are unique and sequential, wrapping around to 0 after reaching the maximum int64 value.
type IDGenerator struct {
	lastID int64
	mutex  sync.Mutex
}

// NewIDGenerator creates and returns a new IDGenerator instance.
// The initial ID is set to 0.
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{lastID: 0}
}

// GetNextID generates and returns the next unique ID.
// It is safe for concurrent use by multiple goroutines.
func (g *IDGenerator) GetNextID() int64 {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if g.lastID == maxInt64 {
		g.lastID = 0
	} else {
		g.lastID++
	}
	return g.lastID
}

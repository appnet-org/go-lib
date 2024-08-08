package plugininterceptor

import (
	"sync"
)

// SharedData is a thread-safe structure to store shared data.
type SharedData struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

func NewSharedData() *SharedData {
	return &SharedData{
		data: make(map[string]interface{}),
	}
}

func (sd *SharedData) Set(key string, value interface{}) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.data[key] = value
}

func (sd *SharedData) Get(key string) (interface{}, bool) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	value, exists := sd.data[key]
	return value, exists
}

package libmetric

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	vmURL       string
	mu          sync.RWMutex
	cacheSeries = make(map[string]*Series)
)

type Series struct {
	key, name string
	labels    []string
	data      atomic.Uint64
	changed   bool
}

func makeKey(name string, labels []string) string {
	return name + "{" + strings.Join(labels, ",") + "}"
}

func MakeSeries(name string, labels ...string) (*Series, error) {
	key := makeKey(name, labels)

	// fast path
	mu.RLock()
	c, ok := cacheSeries[key]
	mu.RUnlock()

	if ok {
		return c, nil
	}

	c = &Series{
		key:    key,
		name:   name,
		labels: labels,
	}

	var (
		currentValue float64 = 0.0
		err          error
	)

	logger.Debug(fmt.Sprintf("[Get] metric %s (%d labels) loaded with value %f", name, len(labels)/2, c.Value()))
	currentValue, err = ReadMetric(name, labels...)
	if err != nil {
		return nil, err
	}

	c.data.Store(f64ToU64(currentValue))

	mu.Lock()
	cacheSeries[key] = c
	mu.Unlock()

	return c, nil
}

func (c *Series) Value() float64 {
	return u64ToF64(c.data.Load())
}

func (c *Series) AddOne() {
	for {
		old := c.data.Load()
		newVal := u64ToF64(old) + 1

		if c.data.CompareAndSwap(old, f64ToU64(newVal)) {
			c.changed = true
			return
		}
	}
}

func (c *Series) Add(x float64) {
	for {
		old := c.data.Load()
		newVal := u64ToF64(old) + x

		if c.data.CompareAndSwap(old, f64ToU64(newVal)) {
			c.changed = true
			return
		}
	}
}

func (c *Series) Set(x float64) {
	for {
		old := c.data.Load()

		// overwrite instead of add
		newBits := f64ToU64(x)

		if c.data.CompareAndSwap(old, newBits) {
			c.changed = true
			return
		}
	}
}

func (c *Series) Commit() error {
	if !c.changed {
		return nil
	}

	err := WriteMetric(c.name, c.Value(), c.labels...)
	if err != nil {
		logger.Error("[Series] Commit failed with error", "err", err)
		c.changed = false
	}
	return err
}

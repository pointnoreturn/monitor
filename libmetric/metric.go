package libmetric

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	vmURL    = os.Getenv("VICTORIA_METRICS")
	mu       sync.RWMutex
	counters = make(map[string]*Counter)
)

type Counter struct {
	name   string
	labels string

	value atomic.Uint64 // stores float64 bits
}

func makeKey(name string, labels []string) string {
	return name + "{" + strings.Join(labels, ",") + "}"
}

func GetCounter(reset bool, name string, labels ...string) (*Counter, error) {
	key := makeKey(name, labels)

	// fast path
	mu.RLock()
	c, ok := counters[key]
	mu.RUnlock()

	if ok {
		if reset {
			c.Reset()
		}
		return c, nil
	}

	// create new
	c = &Counter{
		name:   name,
		labels: strings.Join(labels, ","),
	}

	var (
		currentValue float64 = 0.0
		err          error
	)

	if !reset {
		logger.Debug(fmt.Sprintf("[Get] metric %s (%d labels) loaded with value %f", name, len(labels)/2, c.Value()))
		currentValue, err = ReadMetric(name, labels...)
		if err != nil {
			return nil, err
		}
	} else {
		logger.Debug(fmt.Sprintf("[Get] metric %s (%d labels) is reset to 0", name, len(labels)/2))
	}

	c.value.Store(f64ToU64(currentValue))

	// store
	mu.Lock()
	counters[key] = c
	mu.Unlock()

	return c, nil
}

func (c *Counter) Value() float64 {
	return u64ToF64(c.value.Load())
}

func (c *Counter) Reset() {
	c.value.Store(0)
}

func (c *Counter) AddOne() {
	for {
		old := c.value.Load()
		newVal := u64ToF64(old) + 1

		if c.value.CompareAndSwap(old, f64ToU64(newVal)) {
			return
		}
	}
}

func (c *Counter) Add(x float64) {
	for {
		old := c.value.Load()
		newVal := u64ToF64(old) + x

		if c.value.CompareAndSwap(old, f64ToU64(newVal)) {
			return
		}
	}
}

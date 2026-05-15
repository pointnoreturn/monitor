package libmetric

import (
	"fmt"
	"time"
)

type AutoWrite struct {
	Reset         bool
	Name          string
	WriteInterval time.Duration
	saveTime      time.Time
	c             *Counter
}

func (a *AutoWrite) AddOne(labels ...string) bool {
	return a.Add(1.0, labels...)
}

func (a *AutoWrite) Add(amount float64, labels ...string) bool {
	if a.c == nil {
		c, err := GetCounter(a.Reset, a.Name, labels...)
		if err != nil {
			logger.Error(fmt.Sprintf("[MetricIncrease] Cannot Get %s with %d labels to increase by %f: %v", a.Name, len(labels)/2, amount, err))
			return false
		}
		a.c = c
	}

	a.c.Add(amount)

	if time.Since(a.saveTime) > a.WriteInterval {
		t := time.Now()
		err := WriteMetric(a.Name, a.c.Value(), labels...)
		if err != nil {
			logger.Error(fmt.Sprintf("[MetricIncrease] Cannot WriteMetric %s with %d labels to increase by %f: %v", a.Name, len(labels)/2, amount, err))
			return false
		}
		a.saveTime = t
	}

	return true
}

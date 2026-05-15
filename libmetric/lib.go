package libmetric

import (
	"fmt"
	"log/slog"
)

var (
	serviceUrl string
	logger     *slog.Logger
)

func Init(url string, log *slog.Logger) {
	serviceUrl = url
	logger = log
}

func Increase(amount float64, name string, labels ...string) bool {
	c, err := GetCounter(false, name, labels...)
	if err != nil {
		logger.Error(fmt.Sprintf("[MetricIncrease] Cannot Get %s with %d labels to increase by %f: %v", name, len(labels)/2, amount, err))
		return false
	}

	c.Add(amount)

	return true
}

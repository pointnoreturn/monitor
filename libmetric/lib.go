package libmetric

import (
	"errors"
	"fmt"
	"log/slog"
)

var (
	serviceUrl string
	logger     *slog.Logger

	ErrNoService error = errors.New("serviceUrl was not initialized with libmetric.Init()")
)

func Init(url string, log *slog.Logger) {
	serviceUrl = url
	logger = log
}

func AddOne(name string, labels ...string) bool {
	c, err := MakeSeries(name, labels...)
	if err != nil {
		logger.Error(fmt.Sprintf("[MetricIncrease] Cannot GetCounter %s (%d labels) to increase by 1: %v", name, len(labels)/2, err))
		return false
	}

	c.AddOne()
	c.Commit()

	return true
}

func Add(amount float64, name string, labels ...string) bool {
	c, err := MakeSeries(name, labels...)
	if err != nil {
		logger.Error(fmt.Sprintf("[MetricIncrease] Cannot GetCounter %s (%d labels) to increase by %f: %v", name, len(labels)/2, amount, err))
		return false
	}

	c.Add(amount)
	c.Commit()

	return true
}

func Set(x float64, name string, labels ...string) bool {
	c, err := MakeSeries(name, labels...)
	if err != nil {
		logger.Error(fmt.Sprintf("[MetricIncrease] Cannot GetCounter %s (%d labels) to increase by %f: %v", name, len(labels)/2, x, err))
		return false
	}

	c.Set(x)
	c.Commit()

	return true
}

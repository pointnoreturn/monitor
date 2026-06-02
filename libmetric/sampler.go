package libmetric

import (
	"sort"
	"sync"

	"gonum.org/v1/gonum/stat"
)

type SampleF func([]float64) float64

var (
	Average SampleF = func(row []float64) float64 {
		if len(row) == 0 {
			return 0
		}
		return stat.Mean(row, nil)
	}

	MedianP50 SampleF = func(row []float64) float64 {
		if len(row) == 0 {
			return 0
		}

		cp := append([]float64(nil), row...)
		sort.Float64s(cp)

		return stat.Quantile(0.5, stat.Empirical, cp, nil)
	}

	Median SampleF = func(row []float64) float64 {
		n := len(row)
		if n == 0 {
			return 0
		}

		// copy to avoid mutating input
		cp := make([]float64, n)
		copy(cp, row)

		sort.Float64s(cp)

		mid := n / 2

		if n%2 == 0 {
			return (cp[mid-1] + cp[mid]) / 2
		}

		return cp[mid]
	}

	Sum SampleF = func(row []float64) float64 {
		var s float64
		for i := range row {
			s += row[i]
		}
		return s
	}

	Min SampleF = func(row []float64) float64 {
		if len(row) == 0 {
			return 0
		}
		m := row[0]
		for i := 1; i < len(row); i++ {
			if row[i] < m {
				m = row[i]
			}
		}
		return m
	}

	Max SampleF = func(row []float64) float64 {
		if len(row) == 0 {
			return 0
		}
		m := row[0]
		for i := 1; i < len(row); i++ {
			if row[i] > m {
				m = row[i]
			}
		}
		return m
	}
)

// TODO: Rename, use with Set() as interface?
type Sampler struct {
	Name     string
	MinCount int
	MaxCount int
	Function SampleF

	mu      sync.Mutex
	samples map[string][]float64
}

func (s *Sampler) ensureDefaults() {
	if s.MaxCount <= 0 {
		s.MaxCount = 100
	}
	if s.Function == nil {
		s.Function = Sum
	}
	if s.samples == nil {
		s.samples = make(map[string][]float64)
	}
}

func (s *Sampler) Sample(x float64, labels ...string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureDefaults()

	c, err := MakeSeries(s.Name, labels...)
	if err != nil {
		logger.Error("[Sampler] Sample error", "err", err)
		return false
	}

	// safely append without aliasing issues
	old := s.samples[c.key]
	cp := make([]float64, len(old)+1)
	copy(cp, old)
	cp[len(old)] = x
	s.samples[c.key] = cp

	// bounded window (stable truncation)
	if len(cp) > s.MaxCount {
		s.samples[c.key] = cp[len(cp)-s.MaxCount:]
		cp = s.samples[c.key]
	}

	// only emit when ready
	if s.MinCount > 0 && len(cp) < s.MinCount {
		return true
	}

	value := s.Function(cp)
	c.Set(value)

	return true
}

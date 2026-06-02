package libmetric

import (
	"fmt"
	"time"
)

// TODO: Remove
type Group struct {
	Interval      time.Duration
	pendingWrites map[string]*Series
}

func (g *Group) AddOne(a *AutoCommit, labels ...string) bool {
	s, err := MakeSeries(a.Name, labels...)
	if err != nil {
		logger.Error("[Group] AddOne error", "err", err)
		return false
	}

	s.AddOne()

	if g.pendingWrites == nil {
		g.pendingWrites = make(map[string]*Series)
	}
	g.pendingWrites[s.key] = s

	return true
}

func (g *Group) Add(a *AutoCommit, x float64, labels ...string) bool {
	s, err := MakeSeries(a.Name, labels...)
	if err != nil {
		logger.Error("[Group] Add error", "err", err)
		return false
	}

	s.Add(x)

	if g.pendingWrites == nil {
		g.pendingWrites = make(map[string]*Series)
	}
	g.pendingWrites[s.key] = s

	return true
}

func (g *Group) Update(a *AutoCommit, x float64, labels ...string) bool {
	s, err := MakeSeries(a.Name, labels...)
	if err != nil {
		logger.Error("[Group] Set error", "err", err)
		return false
	}

	s.Set(x)

	if g.pendingWrites == nil {
		g.pendingWrites = make(map[string]*Series)
	}
	g.pendingWrites[s.key] = s

	return true
}

func (g *Group) Sample(a *Sampler, x float64, labels ...string) bool {
	s, err := MakeSeries(a.Name, labels...)
	if err != nil {
		logger.Error("[Group] Sample error", "err", err)
		return false
	}

	a.Sample(x)

	if g.pendingWrites == nil {
		g.pendingWrites = make(map[string]*Series)
	}
	g.pendingWrites[s.key] = s

	return true
}

func (g *Group) Commit() bool {
	var errs []error

	if len(g.pendingWrites) == 0 {
		logger.Debug("[Group] Nothing to commit")
		return true
	}

	leftovers := make(map[string]*Series)

	for _, s := range g.pendingWrites {
		logger.Debug(fmt.Sprintf("[Group] Commit %s", s.name))
		err := s.Commit()
		if err != nil {
			errs = append(errs, err)
			leftovers[s.key] = s
		}
	}

	g.pendingWrites = leftovers

	if len(errs) != 0 {
		logger.Warn(fmt.Sprintf("Have %d errors in commiting group", len(errs)))
	}

	return len(errs) == 0
}

func (g *Group) Ticker() *time.Ticker {
	if g.Interval == 0 {
		g.Interval = time.Second
	}
	return time.NewTicker(g.Interval)
}

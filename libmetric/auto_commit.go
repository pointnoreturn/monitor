package libmetric

// TODO: Replace with multi-commit task
type AutoCommit struct {
	Name string
}

func (g *AutoCommit) AddOne(labels ...string) bool {
	s, err := MakeSeries(g.Name, labels...)
	if err != nil {
		logger.Error("[AutoCommit] AddOne failed", "name", g.Name, "err", err)
		return false
	}

	s.AddOne()

	if err := s.Commit(); err != nil {
		logger.Error("[AutoCommit] AddOne failed (2)", "name", g.Name, "err", err)
		return false
	}

	return true
}

func (g *AutoCommit) Add(x float64, labels ...string) bool {
	s, err := MakeSeries(g.Name, labels...)
	if err != nil {
		logger.Error("[AutoCommit] Add failed", "name", g.Name, "err", err)
		return false
	}

	s.Add(x)

	if err := s.Commit(); err != nil {
		logger.Error("[AutoCommit] Add failed (2)", "name", g.Name, "err", err)
		return false
	}

	return true
}

func (g *AutoCommit) Update(x float64, labels ...string) bool {
	s, err := MakeSeries(g.Name, labels...)
	if err != nil {
		logger.Error("[AutoCommit] Set failed", "name", g.Name, "err", err)
		return false
	}

	s.Set(x)

	if err := s.Commit(); err != nil {
		logger.Error("[AutoCommit] Set failed (2)", "name", g.Name, "err", err)
		return false
	}

	return true
}

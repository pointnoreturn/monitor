package libmetric

type AutoCommit struct {
	Name string
}

func (g *AutoCommit) AddOne(labels ...string) bool {
	s, err := MakeSeries(g.Name, labels...)
	if err != nil {
		// todo log
		return false
	}

	s.AddOne()

	if err := s.Commit(); err != nil {
		return false
	}

	return true
}

func (g *AutoCommit) Add(x float64, labels ...string) bool {
	s, err := MakeSeries(g.Name, labels...)
	if err != nil {
		// todo log
		return false
	}

	s.Add(x)

	if err := s.Commit(); err != nil {
		return false
	}

	return true
}

func (g *AutoCommit) Set(x float64, labels ...string) bool {
	s, err := MakeSeries(g.Name, labels...)
	if err != nil {
		// todo log
		return false
	}

	s.Set(x)

	if err := s.Commit(); err != nil {
		return false
	}

	return true
}

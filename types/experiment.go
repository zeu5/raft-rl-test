package types

import "fmt"

type Experiment struct {
	config *AgentConfig
	name   string
	Result []*Trace
}

func NewExperiment(name string, config *AgentConfig) *Experiment {
	return &Experiment{
		config: config,
		name:   name,
		Result: make([]*Trace, 0),
	}
}

func (e *Experiment) Run() {
	fmt.Printf("Running Experiment: %s\n", e.name)
	agent := NewAgent(e.config)
	for i := 0; i < e.config.Episodes; i++ {
		fmt.Printf("\rExperiment: %s, Episode: %d/%d", e.name, i+1, e.config.Episodes)
		agent.traces[i] = agent.runEpisode()
	}
	e.Result = agent.traces
	fmt.Printf("Finished Experiment: %s\n", e.name)
}

type DataSet interface{}

type Analyzer func([]*Trace) DataSet

type Comparator func([]string, []DataSet)

type Comparison struct {
	Experiments []*Experiment
	analyzer    Analyzer
	comparator  Comparator
}

func NewComparison(analyzer Analyzer, comparator Comparator) *Comparison {
	return &Comparison{
		Experiments: make([]*Experiment, 0),
		analyzer:    analyzer,
		comparator:  comparator,
	}
}

func (c *Comparison) AddExperiment(e *Experiment) {
	c.Experiments = append(c.Experiments, e)
}

func (c *Comparison) Run() {
	datasets := make([]DataSet, len(c.Experiments))
	names := make([]string, len(c.Experiments))
	for i, e := range c.Experiments {
		e.Run()
		datasets[i] = c.analyzer(e.Result)
		names[i] = e.name
	}
	c.comparator(names, datasets)
}

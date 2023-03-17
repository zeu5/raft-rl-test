package rl

import "fmt"

type Experiment struct {
	config          *AgentConfig
	name            string
	Result          []*Trace
	Properties      []*Monitor
	PropertiesStats []int
}

func NewExperiment(name string, config *AgentConfig) *Experiment {
	return &Experiment{
		config:          config,
		name:            name,
		Result:          make([]*Trace, 0),
		Properties:      make([]*Monitor, 0),
		PropertiesStats: make([]int, 0),
	}
}

func NewExperimentWithProperties(name string, config *AgentConfig, properties []*Monitor) *Experiment {
	e := NewExperiment(name, config)
	e.Properties = properties
	e.PropertiesStats = make([]int, len(properties))
	return e
}

func (e *Experiment) hasProperties() bool {
	return len(e.Properties) != 0
}

func (e *Experiment) Run() {
	fmt.Printf("Running Experiment: %s\n", e.name)
	agent := NewAgent(e.config)
	for i := 0; i < e.config.Episodes; i++ {
		fmt.Printf("\rExperiment: %s, Episode: %d/%d", e.name, i+1, e.config.Episodes)
		trace := agent.runEpisode(i)
		agent.traces[i] = trace
		if e.hasProperties() {
			for i, prop := range e.Properties {
				if _, ok := prop.Check(trace); ok {
					e.PropertiesStats[i] += 1
				}
			}
		}
	}
	e.Result = agent.traces
	fmt.Println("")
	if e.hasProperties() {
		for i, count := range e.PropertiesStats {
			fmt.Printf("Property %d satisfied in %d episodes\n", i+1, count)
		}
	}
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

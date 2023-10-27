package types

import (
	"fmt"
)

// Experiment encapsulates the different parameters to configure an agent and analyze the traces
type Experiment struct {
	config *AgentConfig
	name   string
	// Result contains the traces of the experiment
	Result []*Trace
	// Properties to test on the trace
	Properties []*Monitor
	// Number of time each property was satisfied
	PropertiesStats []int
}

// NewExperiment creates a new experiment instance
func NewExperiment(name string, config *AgentConfig) *Experiment {
	return &Experiment{
		config:          config,
		name:            name,
		Result:          make([]*Trace, 0),
		Properties:      make([]*Monitor, 0),
		PropertiesStats: make([]int, 0),
	}
}

// NewExperimentWithProperties creates a new experiment instance with the specified properties
func NewExperimentWithProperties(name string, config *AgentConfig, properties []*Monitor) *Experiment {
	e := NewExperiment(name, config)
	e.Properties = properties
	e.PropertiesStats = make([]int, len(properties))
	return e
}

func (e *Experiment) hasProperties() bool {
	return len(e.Properties) != 0
}

// Run the experiment for the specified number of episodes
// Additionally, for each iteration, check if any of the properties have been satisfied
func (e *Experiment) Run() {
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

// Reset cleans the information about the traces (to save memory)
func (e *Experiment) Reset() {
	e.Result = make([]*Trace, 0)
	e.config.Environment.Reset()
	e.config.Policy.Reset()
}

// Generic Dataset that contains information after processing the traces
type DataSet interface{}

// Analyzer compresses the information in the traces to a DataSet
type Analyzer func(int, string, []*Trace) DataSet

// Comparator differentiates between different datasets with associated names
type Comparator func(int, []string, []DataSet)

// Comparison contains the different experiments to compare
// The traces obtained from the experiments are analyzed
// The analyzed datasets are then compared
type Comparison struct {
	Experiments []*Experiment
	analyzers   map[string]Analyzer
	comparators map[string]Comparator
	runs        int
}

// NewComparison creates a comparison instance
func NewComparison(runs int) *Comparison {
	return &Comparison{
		Experiments: make([]*Experiment, 0),
		analyzers:   make(map[string]Analyzer),
		comparators: make(map[string]Comparator),
		runs:        runs,
	}
}

func (c *Comparison) AddAnalysis(name string, analyzer Analyzer, comparator Comparator) {
	c.analyzers[name] = analyzer
	c.comparators[name] = comparator
}

// Add experiments to compare
func (c *Comparison) AddExperiment(e *Experiment) {
	c.Experiments = append(c.Experiments, e)
}

// Run each experiment sequentially
// TODO: Could be parallelized
func (c *Comparison) Run() {
	for run := 0; run < c.runs; run++ { // number of runs
		fmt.Printf("Run %d\n", run+1)
		datasets := make(map[string][]DataSet) // array with initial capacity - arrayList

		for name := range c.analyzers {
			datasets[name] = make([]DataSet, len(c.Experiments))
		}

		names := make([]string, len(c.Experiments))
		for i, e := range c.Experiments { // index - experiment  in the list of experiments
			e.Run() // running the algorithm, stores the results
			for name, a := range c.analyzers {
				datasets[name][i] = a(run, e.name, e.Result) // call the analyzer on the experiment results
			}
			names[i] = e.name // policy/experiment name
			e.Reset()         //
		}
		for name, comp := range c.comparators {
			comp(run, names, datasets[name]) // make the plots
		}
	}
}

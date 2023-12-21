package types

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
)

// Experiment encapsulates the different parameters to configure an agent and analyze the traces
type Experiment struct {
	*AgentConfig
	Name string
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
		AgentConfig:     config,
		Name:            name,
		Result:          make([]*Trace, 0),
		Properties:      make([]*Monitor, 0),
		PropertiesStats: make([]int, 0),
	}
}

// NewExperimentWithProperties creates a new experiment instance with the specified properties
func NewExperimentWithProperties(name string, config *AgentConfig, properties []*Monitor, record bool) *Experiment {
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
func (e *Experiment) Run(ctx context.Context) {
	agent := NewAgent(e.AgentConfig)
	// episodeTimes := make([]time.Duration, 0)
	for i := 0; i < e.Episodes; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		fmt.Printf("\rExperiment: %s, Episode: %d/%d", e.Name, i+1, e.Episodes)
		// start := time.Now()
		trace := agent.runEpisode(i, ctx)
		// dur := time.Since(start)
		// episodeTimes = append(episodeTimes, dur)
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
	// sumEpisodeTime := time.Duration(0)
	// for _, d := range episodeTimes {
	// 	sumEpisodeTime += d
	// }
	// avgEpisodeTime := time.Duration(int(sumEpisodeTime) / len(episodeTimes))
	// fmt.Printf("\nAverage episode time: %s\n", avgEpisodeTime.String())
	fmt.Println("")
	if e.hasProperties() {
		for i, count := range e.PropertiesStats {
			fmt.Printf("Property %d satisfied in %d episodes\n", i+1, count)
		}
	}
}

func (e *Experiment) Record(run int, basePath string) {
	tracesRecordPath := path.Join(basePath, "traces_"+strconv.Itoa(run)+"_"+e.Name+".jsonl")
	tracesData := new(bytes.Buffer)
	for _, trace := range e.Result {
		bs, err := json.Marshal(trace)
		if err == nil {
			tracesData.Write(bs)
			tracesData.Write([]byte("\n"))
		}
	}
	os.WriteFile(tracesRecordPath, tracesData.Bytes(), 0644)

	policyRecordPath := path.Join(basePath, "policy_"+strconv.Itoa(run)+"_"+e.Name)
	e.Policy.Record(policyRecordPath)
}

// Reset cleans the information about the traces (to save memory)
func (e *Experiment) Reset(run int) {
	e.Result = make([]*Trace, 0)
	e.Environment.Reset()
	e.Policy.Reset()
}

// Generic Dataset that contains information after processing the traces
type DataSet interface{}

// Analyzer compresses the information in the traces to a DataSet
type Analyzer func(int, string, []*Trace) DataSet

// Experiment analyzer takes the whole experiment object for analysis
type EAnalyzer func(int, *Experiment)

// Comparator differentiates between different datasets with associated names
type Comparator func(int, []string, []DataSet)

func NoopComparator() Comparator {
	return func(i int, s []string, ds []DataSet) {}
}

// Comparison contains the different experiments to compare
// The traces obtained from the experiments are analyzed
// The analyzed datasets are then compared
type Comparison struct {
	Experiments []*Experiment
	analyzers   map[string]Analyzer
	eAnalyzers  []EAnalyzer
	comparators map[string]Comparator
	runs        int
	recordPath  string
	record      bool
}

// NewComparison creates a comparison instance
func NewComparison(runs int, recordPath string, record bool) *Comparison {
	return &Comparison{
		Experiments: make([]*Experiment, 0),
		analyzers:   make(map[string]Analyzer),
		eAnalyzers:  make([]EAnalyzer, 0),
		comparators: make(map[string]Comparator),
		runs:        runs,
		recordPath:  recordPath,
		record:      record,
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

func (c *Comparison) AddEAnalysis(ea EAnalyzer) {
	c.eAnalyzers = append(c.eAnalyzers, ea)
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
			e.Run(context.TODO()) // running the algorithm, stores the results
			for name, a := range c.analyzers {
				datasets[name][i] = a(run, e.Name, e.Result) // call the analyzer on the experiment results
			}
			names[i] = e.Name // policy/experiment name
			for _, ea := range c.eAnalyzers {
				ea(run, e)
			}
			if c.record {
				e.Record(run, c.recordPath)
			}
			e.Reset(run) //
		}
		for name, comp := range c.comparators {
			comp(run, names, datasets[name]) // make the plots
		}
	}
}

func (c *Comparison) RunWithCtx(ctx context.Context) {
	for run := 0; run < c.runs; run++ { // number of runs
		fmt.Printf("Run %d\n", run+1)
		datasets := make(map[string][]DataSet) // array with initial capacity - arrayList

		for name := range c.analyzers {
			datasets[name] = make([]DataSet, len(c.Experiments))
		}

		names := make([]string, len(c.Experiments))
		for i, e := range c.Experiments { // index - experiment  in the list of experiments
			select {
			case <-ctx.Done():
				return
			default:
			}
			e.Run(ctx) // running the algorithm, stores the results
			for name, a := range c.analyzers {
				datasets[name][i] = a(run, e.Name, e.Result) // call the analyzer on the experiment results
			}
			names[i] = e.Name // policy/experiment name
			for _, ea := range c.eAnalyzers {
				ea(run, e)
			}
			if c.record {
				e.Record(run, c.recordPath)
			}
			e.Reset(run) //
		}
		for name, comp := range c.comparators {
			comp(run, names, datasets[name]) // make the plots
		}
	}
}

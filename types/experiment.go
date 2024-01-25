package types

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/zeu5/raft-rl-test/util"
)

type experimentRunConfig struct {
	// execution configuration
	CurrentRun int
	Episodes   int
	Horizon    int
	Analyzers  []Analyzer
	Timeout    time.Duration
	Context    context.Context

	// record flags
	RecordTraces bool
	RecordTimes  bool
	RecordPolicy bool

	// last traces configuration
	PrintLastTraces     int
	PrintLastTracesFunc func(*Trace) string

	// reports configuration
	ReportsPrintConfig *ReportsPrintConfig
	ReportSavePath     string
}

// Experiment encapsulates the different parameters to configure an agent and analyze the traces
type Experiment struct {
	Name        string
	policy      Policy
	environment Environment
}

// NewExperiment creates a new experiment instance
func NewExperiment(name string, policy Policy, environment Environment) *Experiment {
	return &Experiment{
		Name:        name,
		policy:      policy,
		environment: environment,
	}
}

func (e *Experiment) recordTrace(rConfig *experimentRunConfig, trace *Trace) {
	tracesFile := path.Join(rConfig.ReportSavePath, "traces", e.Name+"_"+strconv.Itoa(rConfig.CurrentRun)+".jsonl")
	bs, err := json.Marshal(trace)
	if err != nil {
		panic(err)
	}

	util.AppendToFile(tracesFile, string(bs))
}

// Run the experiment for the specified number of episodes
// Additionally, for each iteration, check if any of the properties have been satisfied
func (e *Experiment) Run(rConfig *experimentRunConfig) {
	select {
	case <-rConfig.Context.Done():
		return
	default:
	}

	if rConfig.RecordTraces {
		tracesFolder := path.Join(rConfig.ReportSavePath, "traces")
		if _, err := os.Stat(tracesFolder); err != nil {
			os.MkdirAll(tracesFolder, os.ModePerm)
		}
	}

	totalTimeout := 0
	totalWithError := 0
	consecutiveTimeouts := 0
	consecutiveErrors := 0
	episodeTimes := make([]time.Duration, 0)

	agent := NewAgent(&AgentConfig{
		Episodes:    rConfig.Episodes,
		Horizon:     rConfig.Horizon,
		Policy:      e.policy,
		Environment: e.environment,
	})

	printTracesIndex := rConfig.Episodes - rConfig.PrintLastTraces

	for i := 0; i < rConfig.Episodes; i++ {
		select {
		case <-rConfig.Context.Done():
			return
		default:
		}

		fmt.Printf("\rExperiment: %s, Episode: %d/%d, Timed out: %d, With Error: %d", e.Name, i+1, rConfig.Episodes, totalTimeout, totalWithError)

		eCtx := NewEpisodeContext(i, e.Name, rConfig)
		e.runEpisode(eCtx, agent)
		episodeTimes = append(episodeTimes, eCtx.RunDuration)

		if eCtx.TimedOut {
			totalTimeout += 1
			consecutiveTimeouts += 1
		} else {
			consecutiveTimeouts = 0
		}

		if eCtx.Err != nil {
			totalWithError += 1
			consecutiveErrors += 1
		} else {
			consecutiveErrors = 0
		}

		if rConfig.RecordTraces {
			e.recordTrace(rConfig, eCtx.Trace)
		}

		if !eCtx.TimedOut && eCtx.Err == nil {
			for _, a := range rConfig.Analyzers {
				a.Analyze(rConfig.CurrentRun, i, e.Name, eCtx.Trace)
			}
		}

		if i >= printTracesIndex {
			readableTrace := rConfig.PrintLastTracesFunc(eCtx.Trace)
			filePath := path.Join(rConfig.ReportSavePath, "lastTraces", e.Name+"_run"+strconv.Itoa(rConfig.CurrentRun)+"_ep"+strconv.Itoa(i)+".txt")
			util.WriteToFile(filePath, readableTrace)
		}

		if len(episodeTimes) == 10 {
			if rConfig.RecordTimes {
				e.printEpTimesMs(episodeTimes, rConfig.ReportSavePath)
				e.printEpTimesS(episodeTimes, rConfig.ReportSavePath)
			}
			episodeTimes = make([]time.Duration, 0)
		}

		if consecutiveTimeouts == 10 {
			fmt.Printf("\n Aborting experiment %s : 10 consecutive timeouts\n", e.Name)
			break
		}

		if consecutiveErrors == 10 {
			fmt.Printf("\n Aborting experiment %s : 10 consecutive errors\n", e.Name)
			break
		}
	}

	select {
	case <-rConfig.Context.Done():
		return
	default:
	}

	if rConfig.RecordPolicy {
		e.policy.Record(path.Join(rConfig.ReportSavePath, "policies", e.Name+"_"+strconv.Itoa(rConfig.CurrentRun)))
	}

	fmt.Println("")
}

func (e *Experiment) runEpisode(eCtx *EpisodeContext, agent *Agent) {
	defer func() {
		if r := recover(); r != nil {
			eCtx.SetError(fmt.Errorf("%v", r))
			return
		}
	}()

	select {
	case <-eCtx.Context.Done():
		return
	default:
	}

	done := make(chan error, 1)

	go func(eCtx *EpisodeContext, agent *Agent) {
		start := time.Now()
		agent.RunEpisode(eCtx)
		duration := time.Since(start)

		eCtx.Report.AddTimeEntry(duration, "return_time", "experiment.runEpisode")
		eCtx.RunDuration = duration

		select {
		case <-eCtx.Context.Done():
			if deadline, ok := eCtx.Context.Deadline(); ok && time.Now().After(deadline) {
				eCtx.SetTimedOut()
				eCtx.RecordReport()
			}
		default:
			if eCtx.Err != nil {
				done <- eCtx.Err
				eCtx.RecordReport()
			} else {
				done <- nil
				if rand.Float32() < eCtx.reportPrintConfig.Sampling {
					eCtx.RecordReport()
				}
			}
		}
	}(eCtx, agent)

	select {
	case <-eCtx.Context.Done():
		// Timeout occurred
		deadline, ok := eCtx.Context.Deadline()
		if ok && time.Now().After(deadline) {
			eCtx.SetTimedOut()
		}
	case <-done:
	}

	eCtx.Cancel()
	close(done)
}

func (e *Experiment) printEpTimesMs(epTimes []time.Duration, basePath string) {
	tMilliseconds := ""
	for _, tm := range epTimes {
		tMilliseconds = fmt.Sprintf("%s%7d, ", tMilliseconds, tm.Milliseconds())
	}
	filePath := path.Join(basePath, "epTimes", e.Name+"_ms.txt")
	util.AppendToFile(filePath, tMilliseconds)
}

func (e *Experiment) printEpTimesS(epTimes []time.Duration, basePath string) {
	tSeconds := ""
	for _, tm := range epTimes {
		tSeconds = fmt.Sprintf("%s%3.3f, ", tSeconds, tm.Seconds())
	}
	filePath := path.Join(basePath, "epTimes", e.Name+"_seconds.txt")
	util.AppendToFile(filePath, tSeconds)
}

// Reset cleans the information about the traces (to save memory)
func (e *Experiment) Reset() {
	e.policy.Reset()
}

// Generic Dataset that contains information after processing the traces
type DataSet interface{}

// Analyzer compresses the information in the traces to a DataSet
type Analyzer interface {
	// Run, episode number, experiment, trace
	Analyze(int, int, string, *Trace)
	// Resulting dataset
	DataSet() DataSet
	// Reset the analyzer
	Reset()
}

// Comparator differentiates between different datasets with associated names
// run, total episodes, experiment names, datasets
type Comparator func(int, int, []string, []DataSet)

func NoopComparator() Comparator {
	return func(i, _ int, s []string, ds []DataSet) {}
}

// ComparisonConfig contains the configuration for the comparison
type ComparisonConfig struct {
	Runs     int // number of runs
	Episodes int // number of episodes
	Horizon  int // number of steps

	RecordPath   string              // path to store the results
	ReportConfig *ReportsPrintConfig // configuration for the reports
	Timeout      time.Duration       // timeout for each episode

	// record flags
	RecordTraces bool
	RecordTimes  bool
	RecordPolicy bool

	// last traces configuration
	PrintLastTraces     int
	PrintLastTracesFunc func(*Trace) string
}

// record the configuration of the comparison
func (c *Comparison) recordConfig() {
	cfg := c.cConfig
	if _, ok := os.Stat(cfg.RecordPath); ok != nil {
		os.MkdirAll(cfg.RecordPath, 0777)
	}

	f, err := os.Create(path.Join(cfg.RecordPath, "comparison_config.json"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	out := make(map[string]interface{})
	out["runs"] = cfg.Runs
	out["episodes"] = cfg.Episodes
	out["horizon"] = cfg.Horizon
	out["record_traces"] = cfg.RecordTraces
	out["record_times"] = cfg.RecordTimes
	out["record_policy"] = cfg.RecordPolicy
	out["print_last_traces"] = cfg.PrintLastTraces
	// out["print_last_traces_func"] = cfg.PrintLastTracesFunc
	out["report_config"] = cfg.ReportConfig
	if cfg.Timeout != 0 {
		out["timeout"] = cfg.Timeout.String()
	}

	experiments := make([]string, 0)
	for _, e := range c.Experiments {
		experiments = append(experiments, e.Name)
	}
	out["experiments"] = experiments

	out["analyzers"] = make([]string, 0)
	for name := range c.analyzers {
		out["analyzers"] = append(out["analyzers"].([]string), name)
	}

	bs, err := json.Marshal(out)
	if err != nil {
		panic(err)
	}
	f.Write(bs)
}

// Comparison contains the different experiments to compare
// The traces obtained from the experiments are analyzed
// The analyzed datasets are then compared
type Comparison struct {
	Experiments []*Experiment
	analyzers   map[string]Analyzer
	comparators map[string]Comparator
	cConfig     *ComparisonConfig
}

// NewComparison creates a comparison instance
func NewComparison(config *ComparisonConfig) *Comparison {

	if _, ok := os.Stat(config.RecordPath); ok == nil {
		os.RemoveAll(config.RecordPath)
	}
	os.MkdirAll(config.RecordPath, 0777)

	foldersToCreate := make([]string, 0)

	foldersToCreate = append(foldersToCreate, "epReports")

	if config.RecordTraces {
		foldersToCreate = append(foldersToCreate, "traces")
	}

	if config.RecordTimes {
		foldersToCreate = append(foldersToCreate, "epTimes")
	}

	if config.RecordPolicy {
		foldersToCreate = append(foldersToCreate, "policies")
	}

	if config.PrintLastTraces > 0 {
		if config.PrintLastTracesFunc == nil {
			panic("PrintLastTracesFunc must be defined")
		}
		foldersToCreate = append(foldersToCreate, "lastTraces")
	}

	for _, s := range foldersToCreate {
		fldPath := path.Join(config.RecordPath, s)
		if _, ok := os.Stat(fldPath); ok != nil {
			os.MkdirAll(fldPath, 0777)
		}
	}

	return &Comparison{
		Experiments: make([]*Experiment, 0),
		analyzers:   make(map[string]Analyzer),
		comparators: make(map[string]Comparator),
		cConfig:     config,
	}
}

// AddAnalysis adds an analyzer and comparator to the comparison
func (c *Comparison) AddAnalysis(name string, analyzer Analyzer, comparator Comparator) {
	c.analyzers[name] = analyzer
	c.comparators[name] = comparator
}

// Add experiments to compare
func (c *Comparison) AddExperiment(e *Experiment) {
	c.Experiments = append(c.Experiments, e)
}

// Run the comparison
func (c *Comparison) Run(ctx context.Context) {
	c.recordConfig()

	for run := 0; run < c.cConfig.Runs; run++ { // number of runs
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
			e.Run(c.prepareRunConfig(ctx)) // running the algorithm, stores the results
			for name, a := range c.analyzers {
				datasets[name][i] = a.DataSet() // call the analyzer on the experiment results
				a.Reset()                       // reset the analyzer
			}
			names[i] = e.Name // name of the experiment
			e.Reset()         // reset the experiment
		}
		for name, comp := range c.comparators {
			comp(run, c.cConfig.Episodes, names, datasets[name]) // make the plots
		}
	}
}

// prepare the run configuration for the experiment
func (c *Comparison) prepareRunConfig(ctx context.Context) *experimentRunConfig {
	rCfg := &experimentRunConfig{
		Episodes:            c.cConfig.Episodes,
		Horizon:             c.cConfig.Horizon,
		Analyzers:           make([]Analyzer, 0),
		RecordTraces:        c.cConfig.RecordTraces,
		RecordTimes:         c.cConfig.RecordTimes,
		RecordPolicy:        c.cConfig.RecordPolicy,
		PrintLastTraces:     c.cConfig.PrintLastTraces,
		PrintLastTracesFunc: c.cConfig.PrintLastTracesFunc,
		ReportsPrintConfig:  c.cConfig.ReportConfig,
		ReportSavePath:      c.cConfig.RecordPath,
		Timeout:             c.cConfig.Timeout,
		Context:             ctx,
	}

	for _, a := range c.analyzers {
		rCfg.Analyzers = append(rCfg.Analyzers, a)
	}
	return rCfg
}

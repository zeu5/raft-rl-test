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
	Analyzers  *map[string]*Analyzer
	Timeout    time.Duration
	Context    context.Context

	// thresholds to abort the experiment
	ConsecutiveTimeoutsAbort int
	ConsecutiveErrorsAbort   int

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

	// experiment context
	ExperimentContext *ExperimentContext
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

	//fmt.Printf("Running experiment %s, index %d, parallel index %d - before env SetUp\n", e.Name, rConfig.CurrentRun, rConfig.ExperimentContext.ParallelIndex)
	//time.Sleep(3 * time.Second)
	e.environment.SetUp(rConfig.ExperimentContext.ParallelIndex) // set up the underlying environment based on the parallel index
	//fmt.Printf("Running experiment %s, index %d, parallel index %d - after env SetUp\n", e.Name, rConfig.CurrentRun, rConfig.ExperimentContext.ParallelIndex)

	totalTimeout := 0   // episodes ended with a timeout
	totalWithError := 0 // episodes ended with an error
	consecutiveTimeouts := 0
	consecutiveErrors := 0
	episodeTimes := make([]time.Duration, 0)

	totalOutOfSpaceBounds := 0 // episodes ended with a state out of space bounds
	totalHorizon := 0          // episodes ended with the horizon reached
	totalEpisodes := 0         // total episodes executed
	totalValidEpisodes := 0    // total episodes executed without errors or timeouts (used for RL updates)
	totalValidTimesteps := 0   // total timesteps executed without errors or timeouts (used for RL updates)

	agent := NewAgent(&AgentConfig{
		Episodes:    rConfig.Episodes,
		Horizon:     rConfig.Horizon,
		Policy:      e.policy,
		Environment: e.environment,
	})

	//fmt.Printf("Running experiment %s, index %d, parallel index %d - after agent creation\n", e.Name, rConfig.CurrentRun, rConfig.ExperimentContext.ParallelIndex)

	printTracesIndex := (rConfig.Episodes - rConfig.PrintLastTraces) * rConfig.Horizon // compute the index to print the last N timesteps
	executedTimesteps := 0
	availableTimesteps := rConfig.Episodes * rConfig.Horizon

	// output paddings
	TSPadding := len(strconv.Itoa(availableTimesteps))
	EPPadding := len(strconv.Itoa(rConfig.Episodes))
	NamePadding := rConfig.ExperimentContext.LongestNameLen

	// terminal execution display
	outPrintable := fmt.Sprintf("Exp:%*s, TSteps:%*d/%d, Valid:%*d [%5.1f%%] || Eps:%*d, Valid:%*d [%5.1f%%], TOut:%*d, Err:%*d || Horizon:%*d, Bound:%*d",
		NamePadding, e.Name, TSPadding, executedTimesteps, availableTimesteps, TSPadding, totalValidTimesteps, (float32(totalValidTimesteps)/float32((executedTimesteps)))*100,
		EPPadding, totalEpisodes, EPPadding, totalValidEpisodes, float32(totalValidEpisodes)/float32(totalEpisodes)*100, EPPadding, totalTimeout, EPPadding, totalWithError,
		EPPadding, totalHorizon, EPPadding, totalOutOfSpaceBounds)

	//fmt.Printf("Running experiment %s, index %d, parallel index %d - before trySet on parallel output\n", e.Name, rConfig.CurrentRun, rConfig.ExperimentContext.ParallelIndex)
	rConfig.ExperimentContext.ParallelOutput.TrySet(outPrintable)
	//fmt.Printf("Running experiment %s, index %d, parallel index %d - after trySet on parallel output\n", e.Name, rConfig.CurrentRun, rConfig.ExperimentContext.ParallelIndex)

	for executedTimesteps < availableTimesteps {
		select {
		case <-rConfig.Context.Done():
			*rConfig.ExperimentContext.CompletedChannel <- ExperimentResult{
				ExperimentIndex:  rConfig.ExperimentContext.ExperimentIndex,
				ParallelRunIndex: rConfig.ExperimentContext.ParallelIndex,

				TotalTimesteps:  executedTimesteps,
				ValidTimesteps:  totalValidTimesteps,
				TotalEpisodes:   totalEpisodes,
				ValidEpisodes:   totalValidEpisodes,
				ErrorEpisodes:   totalWithError,
				TimeoutEpisodes: totalTimeout,
			}
			return
		default:
		}
		//fmt.Printf("Running experiment %s, index %d, parallel index %d - before creating episode context\n", e.Name, rConfig.CurrentRun, rConfig.ExperimentContext.ParallelIndex)

		eCtx := NewEpisodeContext(executedTimesteps, totalEpisodes, e.Name, rConfig) // create a new episode context to store the info used and returned by the episode
		if executedTimesteps >= printTracesIndex {                                   // print the report for the last N episodes (or timestep-wise)
			eCtx.SetToPrintReport(true)
		}

		//fmt.Printf("Running experiment %s, index %d, parallel index %d - created episode context\n", e.Name, rConfig.CurrentRun, rConfig.ExperimentContext.ParallelIndex)

		e.runEpisode(eCtx, agent)                             // run the episode
		episodeTimes = append(episodeTimes, eCtx.RunDuration) // store the episode time for the statistics

		//fmt.Printf("Running experiment %s, index %d, parallel index %d - after running one episode\n", e.Name, rConfig.CurrentRun, rConfig.ExperimentContext.ParallelIndex)

		startingTimesteps := executedTimesteps // store the number of timesteps executed before the episode, used for the analysis
		executedTimesteps += eCtx.Timesteps    // increment the number of valid timesteps executed
		totalEpisodes += 1                     // increment the number of episodes executed

		// possible outcomes of the episode
		// episode timedout
		if eCtx.TimedOut {
			totalTimeout += 1
			consecutiveTimeouts += 1
		} else {
			consecutiveTimeouts = 0
		}

		// episode with error
		if eCtx.Err != nil {
			totalWithError += 1
			consecutiveErrors += 1
		} else {
			consecutiveErrors = 0
		}

		if rConfig.RecordTraces {
			e.recordTrace(rConfig, eCtx.Trace)
		}

		// analyze the trace, even if the episode timed out or ended with an error
		AnalyzerMap := *rConfig.Analyzers
		for _, a := range AnalyzerMap {
			a := *a
			a.Analyze(rConfig.CurrentRun, totalEpisodes, startingTimesteps, e.Name, eCtx.Trace)
		}

		// if no error or timeout, increment the number of valid episodes executed
		if !eCtx.TimedOut && eCtx.Err == nil {
			// increment the number of valid episodes executed and subcategories
			totalValidEpisodes += 1
			totalValidTimesteps += eCtx.Timesteps
			// episode ending category
			if eCtx.OutOfSpaceBounds { // terminal state
				totalOutOfSpaceBounds += 1
			} else if eCtx.HorizonEnd { // horizon reached
				totalHorizon += 1
			}
		}

		// print the last N traces
		if executedTimesteps >= printTracesIndex {
			readableTrace := rConfig.PrintLastTracesFunc(eCtx.Trace)
			filePath := path.Join(rConfig.ReportSavePath, "lastTraces", e.Name+"_run"+strconv.Itoa(rConfig.CurrentRun)+"_ep"+strconv.Itoa(totalEpisodes)+".txt")
			util.WriteToFile(filePath, readableTrace)
		}

		// print episode times
		if len(episodeTimes) == 10 {
			if rConfig.RecordTimes {
				e.printEpTimesMs(episodeTimes, rConfig.ReportSavePath)
				e.printEpTimesS(episodeTimes, rConfig.ReportSavePath)
			}
			episodeTimes = make([]time.Duration, 0)
		}

		// check to eventually abort the experiment
		if consecutiveTimeouts >= rConfig.ConsecutiveTimeoutsAbort {
			fmt.Printf("\n Aborting experiment %s : %d consecutive timeouts\n", e.Name, consecutiveTimeouts)
			*rConfig.ExperimentContext.FailedChannel <- ExperimentResult{
				ExperimentIndex:  rConfig.ExperimentContext.ExperimentIndex,
				ParallelRunIndex: rConfig.ExperimentContext.ParallelIndex,

				TotalTimesteps:  executedTimesteps,
				ValidTimesteps:  totalValidTimesteps,
				TotalEpisodes:   totalEpisodes,
				ValidEpisodes:   totalValidEpisodes,
				ErrorEpisodes:   totalWithError,
				TimeoutEpisodes: totalTimeout,

				FailureLog: fmt.Sprintf("%d consecutive timeouts", consecutiveTimeouts),
			}
			return
		}

		if consecutiveErrors >= rConfig.ConsecutiveErrorsAbort {
			fmt.Printf("\n Aborting experiment %s : %d consecutive errors\n", e.Name, consecutiveErrors)
			*rConfig.ExperimentContext.FailedChannel <- ExperimentResult{
				ExperimentIndex:  rConfig.ExperimentContext.ExperimentIndex,
				ParallelRunIndex: rConfig.ExperimentContext.ParallelIndex,

				TotalTimesteps:  executedTimesteps,
				ValidTimesteps:  totalValidTimesteps,
				TotalEpisodes:   totalEpisodes,
				ValidEpisodes:   totalValidEpisodes,
				ErrorEpisodes:   totalWithError,
				TimeoutEpisodes: totalTimeout,

				FailureLog: fmt.Sprintf("%d consecutive errors", consecutiveErrors),
			}
			return
		}

		// terminal execution display
		outPrintable = fmt.Sprintf("Exp:%*s, TSteps:%*d/%d, Valid:%*d [%5.1f%%] || Eps:%*d, Valid:%*d [%5.1f%%], TOut:%*d, Err:%*d || Horizon:%*d, Bound:%*d",
			NamePadding, e.Name, TSPadding, executedTimesteps, availableTimesteps, TSPadding, totalValidTimesteps, (float32(totalValidTimesteps)/float32((executedTimesteps)))*100,
			EPPadding, totalEpisodes, EPPadding, totalValidEpisodes, float32(totalValidEpisodes)/float32(totalEpisodes)*100, EPPadding, totalTimeout, EPPadding, totalWithError,
			EPPadding, totalHorizon, EPPadding, totalOutOfSpaceBounds)
		rConfig.ExperimentContext.ParallelOutput.TrySet(outPrintable)
	}

	select {
	case <-rConfig.Context.Done():
		*rConfig.ExperimentContext.CompletedChannel <- ExperimentResult{
			ExperimentIndex:  rConfig.ExperimentContext.ExperimentIndex,
			ParallelRunIndex: rConfig.ExperimentContext.ParallelIndex,

			TotalTimesteps:  executedTimesteps,
			ValidTimesteps:  totalValidTimesteps,
			TotalEpisodes:   totalEpisodes,
			ValidEpisodes:   totalValidEpisodes,
			ErrorEpisodes:   totalWithError,
			TimeoutEpisodes: totalTimeout,
		}
		return
	default:
	}

	if rConfig.RecordPolicy {
		e.policy.Record(path.Join(rConfig.ReportSavePath, "policies", e.Name+"_"+strconv.Itoa(rConfig.CurrentRun)))
	}

	// fmt.Println("")
	*rConfig.ExperimentContext.CompletedChannel <- ExperimentResult{
		ExperimentIndex:  rConfig.ExperimentContext.ExperimentIndex,
		ParallelRunIndex: rConfig.ExperimentContext.ParallelIndex,

		TotalTimesteps:  executedTimesteps,
		ValidTimesteps:  totalValidTimesteps,
		TotalEpisodes:   totalEpisodes,
		ValidEpisodes:   totalValidEpisodes,
		ErrorEpisodes:   totalWithError,
		TimeoutEpisodes: totalTimeout,
	}
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
		case <-eCtx.Context.Done(): // episode ended with a timeout
			if deadline, ok := eCtx.Context.Deadline(); ok && time.Now().After(deadline) {
				eCtx.SetTimedOut()
				eCtx.RecordReport()
			}
		default:
			if eCtx.Err != nil { // episode ended with an error
				done <- eCtx.Err
				eCtx.RecordReport()
			} else {
				done <- nil
				if eCtx.Timesteps < agent.config.Horizon { // episode ended because reached terminal state before the horizon
					eCtx.OutOfSpaceBounds = true
				} else { // episode ended because the horizon was reached
					eCtx.HorizonEnd = true
				}
				if eCtx.ToPrintReport {
					eCtx.RecordReport()
				} else if rand.Float32() < eCtx.reportPrintConfig.Sampling {
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
	// Run, total timesteps, experiment, trace
	Analyze(int, int, int, string, *Trace)
	// Resulting dataset
	DataSet() DataSet
	// Reset the analyzer
	Reset()
}

// Comparator differentiates between different datasets with associated names
// run, total episodes, experiment names, datasets
type Comparator func(int, int, []string, map[string]DataSet)

func NoopComparator() Comparator {
	return func(i, _ int, s []string, ds map[string]DataSet) {}
}

// ComparisonConfig contains the configuration for the comparison
type ComparisonConfig struct {
	Runs     int // number of runs
	Episodes int // number of episodes
	Horizon  int // number of steps

	RecordPath   string              // path to store the results
	ReportConfig *ReportsPrintConfig // configuration for the reports
	Timeout      time.Duration       // timeout for each episode

	// thresholds to abort the experiment
	ConsecutiveTimeoutsAbort int
	ConsecutiveErrorsAbort   int

	// record flags
	RecordTraces bool
	RecordTimes  bool
	RecordPolicy bool

	// last traces configuration
	PrintLastTraces     int
	PrintLastTracesFunc func(*Trace) string

	EnvCtor             func() PartitionedSystemEnvironment
	ParallelExperiments int

	TimeBudget time.Duration
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
	out["duration"] = cfg.TimeBudget.String()
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
	for name := range c.analyzerCtors {
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
	Experiments   []*Experiment
	analyzers     map[int]*map[string]*Analyzer // map of analyzers for each experiment: experiment index -> analyzer name -> analyzer
	analyzerCtors map[string]func() Analyzer    // map of analyzer constructors: analyzer name -> constructor
	comparators   map[string]Comparator
	cConfig       *ComparisonConfig
}

// NewComparison creates a comparison instance
func NewComparison(config *ComparisonConfig) *Comparison {

	if _, ok := os.Stat(config.RecordPath); ok == nil {
		// os.RemoveAll(config.RecordPath)
		RemoveContents(config.RecordPath)
	}
	os.MkdirAll(config.RecordPath, 0777)

	foldersToCreate := make([]string, 0)

	foldersToCreate = append(foldersToCreate, "epReports")
	foldersToCreate = append(foldersToCreate, "coverage")

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
		Experiments:   make([]*Experiment, 0),
		analyzers:     make(map[int]*map[string]*Analyzer),
		analyzerCtors: make(map[string]func() Analyzer),
		comparators:   make(map[string]Comparator),
		cConfig:       config,
	}
}

// AddAnalysis adds an analyzer and comparator to the comparison
func (c *Comparison) AddAnalysis(name string, analyzerCtor func() Analyzer, comparator Comparator) {
	c.analyzerCtors[name] = analyzerCtor
	c.comparators[name] = comparator
}

// Add experiments to compare
func (c *Comparison) AddExperiment(e *Experiment) {
	c.Experiments = append(c.Experiments, e)
}

// Run the comparison
func (c *Comparison) Run(ctx context.Context) {
	// fmt.Println("Comparison.Run Start")
	c.recordConfig() // store configuration details to a file

	startTime := time.Now()
	timer := time.NewTimer(c.cConfig.TimeBudget)
	ticker := time.NewTicker(1 * time.Second)

	longestNameLen := 0
	for _, e := range c.Experiments {
		if len(e.Name) > longestNameLen {
			longestNameLen = len(e.Name)
		}
	}

	for run := 0; run < c.cConfig.Runs; run++ { // number of runs
		fmt.Printf("Run %d\n", run+1)
		datasets := make(map[string]map[string]DataSet) // map of datasets for each analyzer: analyzer name -> datasets

		strOutcomes := make(map[string]string, 0)
		completedExpsCount := 0
		failedExpsCount := 0

		for name := range c.analyzerCtors {
			datasets[name] = make(map[string]DataSet, len(c.Experiments))
		}

		expNames := make([]string, 0)

		// completedExp := make([]int, 0) // index of the completed experiments
		nextExpIndex := 0 // index of the next experiment to run
		runningExp := 0   // number of experiments running
		totalRunExp := 0  // total number of experiments run

		completedChannel := make(chan ExperimentResult, c.cConfig.ParallelExperiments) // channel to use by completed exps
		failedChannel := make(chan ExperimentResult, c.cConfig.ParallelExperiments)    // channel to use by failed exps
		endChannel := make(chan struct{})                                              // channel to signal the end of the comparison

		parallelCtxCancels := make([]context.CancelFunc, 0)

		// list of free parallel indexes, to give to an experiment when it starts (output slot, ...)
		freeParallelIndexes := make([]int, c.cConfig.ParallelExperiments)
		for i := 0; i < c.cConfig.ParallelExperiments; i++ {
			freeParallelIndexes[i] = i
		}

		// output, +1 first is for the time
		expOutputs := make([]*ParallelOutput, c.cConfig.ParallelExperiments+1)
		for i := 0; i < c.cConfig.ParallelExperiments+1; i++ {
			expOutputs[i] = NewParallelOutput()
		}

		expOutputs[0].Running = true // set the first output to running
		expOutputs[0].Set("[RUNNING] Time Limit: - --- Time: -")

		printer := NewTerminalPrinter(ctx, &expOutputs, 2)
		printer.Start()

		// fmt.Println("Printer started")
		// time.Sleep(3 * time.Second)

	ExpsLoop:
		for totalRunExp < len(c.Experiments) {
			for runningExp < c.cConfig.ParallelExperiments && nextExpIndex < len(c.Experiments) {
				pIndex := freeParallelIndexes[0]              // get the first free parallel index
				freeParallelIndexes = freeParallelIndexes[1:] // remove the first free parallel index

				// prepare analyzers
				// create the map of analyzers for the experiment
				aMap := make(map[string]*Analyzer)
				c.analyzers[nextExpIndex] = &aMap

				// instantiate and add analyzers to the map
				for name, analyzerConstructor := range c.analyzerCtors {
					analyzer := analyzerConstructor() // create the analyzer
					mp := *c.analyzers[nextExpIndex]  // get the map of analyzers for the experiment
					mp[name] = &analyzer              // add the analyzer to the map
				}

				// create the context for the experiment
				parallelCtx, cancel := context.WithCancel(ctx)
				parallelCtxCancels = append(parallelCtxCancels, cancel)

				// create the context for the experiment
				expContext := &ExperimentContext{
					Context:          parallelCtx,                      // context to cancel the experiment
					ExperimentName:   c.Experiments[nextExpIndex].Name, // name of the experiment
					ExperimentIndex:  nextExpIndex,                     // index of the experiment in the list of experiments
					ParallelIndex:    pIndex,                           // index of the experiment in the parallel slot (for printing and instantiating environment)
					LongestNameLen:   longestNameLen,                   // length of the longest experiment name, used for padding when printing
					ParallelOutput:   expOutputs[pIndex+1],             // where to store the current output of the experiment, +1 because the first slot is for the time
					Analyzers:        c.analyzers[nextExpIndex],        // analyzers to be run on the experiment
					CompletedChannel: &completedChannel,                // channel to signal that the experiment has completed
					FailedChannel:    &failedChannel,                   // channel to signal that the experiment has failed
				}
				expOutputs[pIndex+1].Running = true // set the parallel output to running

				c.startExperiment(c.Experiments[nextExpIndex], expContext)
				runningExp++   // increment the counter of running experiments
				nextExpIndex++ // increment the index of the next experiment to run
				// fmt.Printf("Experiment %d started, running experiments: %d, next experiment index: %d, completed experiments: %v, free parallel indexes: %v\n", nextExpIndex-1, runningExp, nextExpIndex, completedExp, freeParallelIndexes)
			}
			select {
			case <-ctx.Done():
				return
			case completedResult := <-completedChannel:
				runningExp--
				totalRunExp++
				// completedExp = append(completedExp, completedResult.ExperimentIndex)
				freeParallelIndexes = append(freeParallelIndexes, completedResult.ParallelRunIndex)

				eName := c.Experiments[completedResult.ExperimentIndex].Name

				// expOutputs[completedResult.ParallelRunIndex].Running = false // set the parallel output to running
				expOutputs[completedResult.ParallelRunIndex].Set(fmt.Sprintf("Exp: %*s --Completed--", longestNameLen, eName))

				// append to string outcome
				expOutcome := fmt.Sprintf("COMPLETED - Time passed: %s\n", time.Since(startTime).Truncate(time.Second).String())
				expOutcome = fmt.Sprintf("%s%s", expOutcome, completedResult.Printable())
				strOutcomes[eName] = expOutcome

				// store the analyzers results in the datasets
				aMap := *c.analyzers[completedResult.ExperimentIndex]
				expNames = append(expNames, c.Experiments[completedResult.ExperimentIndex].Name)
				for name, a := range aMap {
					a := *a
					datasets[name][c.Experiments[completedResult.ExperimentIndex].Name] = a.DataSet() // store the analyzer results in the datasets - analyzer name -> experiment name -> dataset
				}
				if totalRunExp == len(c.Experiments) { // if all the experiments have completed
					close(endChannel)
				}

			case failedResult := <-failedChannel:
				runningExp--
				totalRunExp++

				freeParallelIndexes = append(freeParallelIndexes, failedResult.ParallelRunIndex)

				eName := c.Experiments[failedResult.ExperimentIndex].Name

				// expOutputs[failedResult.ParallelRunIndex].Running = false // set the parallel output to running
				expOutputs[failedResult.ParallelRunIndex].Set(fmt.Sprintf("Exp: %*s --Failed--", longestNameLen, eName))

				// append to string outcome
				expOutcome := fmt.Sprintf("FAILED: %s - Time passed: %s\n", failedResult.FailureLog, time.Since(startTime).Truncate(time.Second).String())
				expOutcome = fmt.Sprintf("%s%s", expOutcome, failedResult.Printable())
				strOutcomes[eName] = expOutcome

				if totalRunExp == len(c.Experiments) { // if all the experiments have completed
					close(endChannel)
				}

			case <-ticker.C:
				expOutputs[0].Set(fmt.Sprintf("[RUNNING] Time Limit: %s --- Time: %s", c.cConfig.TimeBudget.String(), time.Since(startTime).Truncate(time.Second).String()))

			case <-endChannel:
				fmt.Printf("EndChannel\n")
				break ExpsLoop

			// time is up, cancel all the running experiments and proceed to comparators
			case <-timer.C:
				for _, cancel := range parallelCtxCancels {
					cancel()
				}

			}
		}

		printer.Stop()

		// append to string outcome
		generalOutcome := fmt.Sprintf("Time passed: %s\n", time.Since(startTime).Truncate(time.Second).String())
		generalOutcome = fmt.Sprintf("%sRun %d: Total: %d, Completed: %d, Failed: %d\n", generalOutcome, run, len(expNames), completedExpsCount, failedExpsCount)
		strOutcomes["General"] = generalOutcome

		// print the outcomes
		c.printOutcomes(strOutcomes)

		for name, comp := range c.comparators {
			comp(run, c.cConfig.Episodes, expNames, datasets[name]) // make the plots
		}
	}
}

func (c *Comparison) startExperiment(e *Experiment, eCtx *ExperimentContext) {
	expConfig := c.prepareRunConfig(eCtx)
	go func() {
		e.Run(expConfig) // running the algorithm, stores the results
	}()
}

func (c *Comparison) printOutcomes(outcomes map[string]string) {
	filepath := path.Join(c.cConfig.RecordPath, "expOutcomes.txt")
	result := ""

	result += "General:\n"
	result += outcomes["General"]
	result += "\n"

	for name, outcome := range outcomes {
		if name == "General" {
			continue
		}
		result += "[" + name + "]\n"
		result += outcome
		result += "\n"
	}

	util.WriteToFile(filepath, result)
}

// prepare the run configuration for the experiment
func (c *Comparison) prepareRunConfig(eCtx *ExperimentContext) *experimentRunConfig {
	rCfg := &experimentRunConfig{
		Episodes:            c.cConfig.Episodes,
		Horizon:             c.cConfig.Horizon,
		Analyzers:           eCtx.Analyzers,
		RecordTraces:        c.cConfig.RecordTraces,
		RecordTimes:         c.cConfig.RecordTimes,
		RecordPolicy:        c.cConfig.RecordPolicy,
		PrintLastTraces:     c.cConfig.PrintLastTraces,
		PrintLastTracesFunc: c.cConfig.PrintLastTracesFunc,
		ReportsPrintConfig:  c.cConfig.ReportConfig,
		ReportSavePath:      c.cConfig.RecordPath,
		Timeout:             c.cConfig.Timeout,
		Context:             eCtx.Context,

		ExperimentContext: eCtx,
	}

	if rCfg.ConsecutiveErrorsAbort == 0 {
		rCfg.ConsecutiveErrorsAbort = 10
	}
	if rCfg.ConsecutiveTimeoutsAbort == 0 {
		rCfg.ConsecutiveTimeoutsAbort = 10
	}

	return rCfg
}

// Delete everything in the directory except the outtext.txt file
func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		if name != "outtext.txt" {
			err = os.RemoveAll(path.Join(dir, name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

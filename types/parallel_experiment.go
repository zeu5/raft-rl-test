package types

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gosuri/uilive"
)

type ExperimentContext struct {
	Context context.Context

	ExperimentName  string // name of the experiment
	PrintableStatus string // status of the experiment to be printed to terminal
	ExperimentIndex int    // index of the experiment, to be used to signal which experiment ended or similar
	ParallelIndex   int    // index of the experiment in the parallel slot (for printing)

	LongestNameLen int // length of the longest experiment name, used for padding when printing

	ParallelOutput *ParallelOutput // where to store the current output of the experiment

	Analyzers *map[string]*Analyzer // analyzers to be run on the experiment

	CompletedChannel *chan ExperimentResult // channel to signal that the experiment has completed
	FailedChannel    *chan ExperimentResult // channel to signal that the experiment has failed

	StartTime time.Time
	TimeLimit time.Duration
}

func NewExperimentContext(ctx context.Context, name string, expIndex int, parallelIndex int, longestNameLength int, output *ParallelOutput,
	analyzers *map[string]*Analyzer, completedChannel *chan ExperimentResult, failedChannel *chan ExperimentResult,
	startTime time.Time, timeLimit time.Duration) *ExperimentContext {
	return &ExperimentContext{
		Context: ctx,

		ExperimentName:  name,
		PrintableStatus: "Pending",
		ExperimentIndex: expIndex,
		ParallelIndex:   parallelIndex,

		LongestNameLen: longestNameLength,

		ParallelOutput: output,

		Analyzers: analyzers,

		CompletedChannel: completedChannel,
		FailedChannel:    failedChannel,

		StartTime: startTime,
		TimeLimit: timeLimit,
	}
}

type ExperimentResult struct {
	ExperimentIndex  int // index of the experiment in the experiment list
	ParallelRunIndex int // index of the experiment in the parallel slot (for printing)

	TotalTimesteps  int
	ValidTimesteps  int
	TotalEpisodes   int
	ValidEpisodes   int
	ErrorEpisodes   int
	TimeoutEpisodes int

	FailureLog string
}

func (e ExperimentResult) Printable() string {
	result := ""
	result += fmt.Sprintf("Experiment Index: %d\n", e.ExperimentIndex)

	result += fmt.Sprintf("Total Timesteps: %d\n", e.TotalTimesteps)
	result += fmt.Sprintf("Valid Timesteps: %d [%5.1f%%]\n", e.ValidTimesteps, float64(e.ValidTimesteps)/float64(e.TotalTimesteps)*100)
	result += fmt.Sprintf("Total Episodes: %d\n", e.TotalEpisodes)
	result += fmt.Sprintf("Valid Episodes: %d [%5.1f%%]\n", e.ValidEpisodes, float64(e.ValidEpisodes)/float64(e.TotalEpisodes)*100)
	result += fmt.Sprintf("Error Episodes: %d [%5.1f%%]\n", e.ErrorEpisodes, float64(e.ErrorEpisodes)/float64(e.TotalEpisodes)*100)
	result += fmt.Sprintf("Timeout Episodes: %d [%5.1f%%]\n", e.TimeoutEpisodes, float64(e.TimeoutEpisodes)/float64(e.TotalEpisodes)*100)

	return result
}

// TERMINAL PRINTER

type TerminalPrinter struct {
	parallelOutputs *[]*ParallelOutput
	ctx             context.Context
	printerCtx      context.Context
	printerCancel   context.CancelFunc
	frequency       int

	writer  *uilive.Writer
	writers []io.Writer
}

func NewTerminalPrinter(ctx context.Context, parallelOutputs *[]*ParallelOutput, frequency int) *TerminalPrinter {
	printerCtx, cancel := context.WithCancel(ctx)
	size := len(*parallelOutputs)
	writers := make([]io.Writer, size)
	writer := uilive.New()
	for i := 0; i < size-1; i++ {
		writers[i] = writer.Newline()
	}

	return &TerminalPrinter{
		parallelOutputs: parallelOutputs,
		ctx:             ctx,
		printerCtx:      printerCtx,
		printerCancel:   cancel,
		frequency:       frequency,

		writer:  writer,
		writers: writers,
	}
}

func (p *TerminalPrinter) Start() {
	go func() {
		for {
			select {
			case <-p.printerCtx.Done():
				p.writer.Stop()
				return
			case <-p.ctx.Done():
				p.writer.Stop()
				return
			case <-time.After(time.Duration(p.frequency) * time.Second):
				p.print()
			}
		}
	}()
}

func (p *TerminalPrinter) Stop() {
	p.printerCancel()
}

func (p *TerminalPrinter) print() {
	// fmt.Printf("Printer print start\n")
	for i, output := range *p.parallelOutputs {
		if !output.Running {
			continue
		}
		s := output.Get()

		//fmt.Printf("toPrint: %s\n", s)
		if i == 0 {
			fmt.Fprint(p.writer, s+"\n")

		} else {
			fmt.Fprint(p.writers[i-1], s+"\n")
		}
	}
	p.writer.Flush()
	// fmt.Printf("Printer print end\n")

	// s := "printing\r"
	// fmt.Fprint(p.writer, s)
	// p.writer.Flush()
}

func (p *TerminalPrinter) printDebug() {
	for i, output := range *p.parallelOutputs {
		s := output.Get()
		if s != "" {
			fmt.Printf("output %d: %s\n", i, s)
		}
	}
}

// PARALLEL OUTPUT

// used to update and print experiment outputs
type ParallelOutput struct {
	mu        sync.Mutex
	printable string

	Running bool
}

func NewParallelOutput() *ParallelOutput {
	return &ParallelOutput{
		mu:        sync.Mutex{},
		printable: "",

		Running: false,
	}
}

// Set the output string (blocking)
func (p *ParallelOutput) Set(s string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.printable = s
}

// Try to set the output string (non-blocking)
func (p *ParallelOutput) TrySet(s string) bool {
	success := p.mu.TryLock()
	if success {
		defer p.mu.Unlock()
		p.printable = s
		// fmt.Printf("output set to: %s\n", p.printable)
		return true
	}
	return false
}

// Get the output string (blocking)
func (p *ParallelOutput) Get() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.printable
}

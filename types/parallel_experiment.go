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
}

func NewExperimentContext(ctx context.Context, name string, expIndex int, parallelIndex int, longestNameLength int, output *ParallelOutput, analyzers *map[string]*Analyzer, completedChannel *chan ExperimentResult, failedChannel *chan ExperimentResult) *ExperimentContext {
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
	}
}

type ExperimentResult struct {
	ExperimentIndex  int // index of the experiment in the experiment list
	ParallelRunIndex int // index of the experiment in the parallel slot (for printing)
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

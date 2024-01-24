package explorer

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/policies"
)

type Explorer struct {
	PolicyFile string
	TracesFile string

	QTable *policies.QTable
	Traces []*Trace

	StateMap map[string]*State
}

// Create an explorer of q tables and trace
func NewExplorer(policyFile string, tracesFile string) (*Explorer, error) {
	e := &Explorer{
		PolicyFile: policyFile,
		TracesFile: tracesFile,
		QTable:     policies.NewQTable(),
		Traces:     make([]*Trace, 0),
		StateMap:   make(map[string]*State),
	}

	err := e.QTable.Read(policyFile)
	if err != nil {
		return nil, err
	}
	e.Traces, err = readTraces(e.TracesFile)
	if err != nil {
		return nil, err
	}

	for _, t := range e.Traces {
		for _, s := range t.States {
			if _, ok := e.StateMap[s.Key]; !ok {
				e.StateMap[s.Key] = s
			}
		}
	}

	return e, nil
}

func readTraces(path string) ([]*Trace, error) {
	traces := make([]*Trace, 0)
	file, err := os.Open(path)
	if err != nil {
		return traces, fmt.Errorf("error reading file: %s", err)
	}
	defer file.Close()

	if strings.Contains(path, "json") && !strings.Contains(path, "jsonl") {
		t := NewTrace()
		data, err := io.ReadAll(file)
		if err != nil {
			return traces, fmt.Errorf("error reading file: %s", err)
		}
		err = json.Unmarshal(data, t)
		if err != nil {
			return traces, fmt.Errorf("error parsing file: %s", err)
		}
		traces = append(traces, t)
	} else if strings.Contains(path, "jsonl") {
		scanner := bufio.NewScanner(file)
		maxTraceSize := 5 * 1024 * 1024
		scanner.Buffer(make([]byte, maxTraceSize), maxTraceSize)
		for scanner.Scan() {
			t := NewTrace()
			bs := scanner.Bytes()
			if len(bs) >= maxTraceSize {
				return traces, errors.New("error trace too big")
			}
			err := json.Unmarshal(bs, t)
			if err != nil {
				return traces, fmt.Errorf("error reading file contents: %s", err)
			}
			if len(t.States) != len(t.Actions) || len(t.Actions) != len(t.NextStates) || len(t.States) != len(t.NextStates) {
				return traces, fmt.Errorf("number of states, actions and next states mismatched")
			}
			traces = append(traces, t)
		}
		if scanner.Err() != nil {
			return traces, fmt.Errorf("failed to read traces: %s", err)
		}
	}
	return traces, nil
}

// Example invocation - ./raft-rl-test explore [policy_output(.jsonl)] [trace_output(.jsonl)]
func ExploreCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "explore [policy_output] [trace_output]",
		Long: "Explore the choices of a q-table and the traces",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			exp, err := NewExplorer(args[0], args[1])
			if err != nil {
				return err
			}

			exp.Interact()
			return nil
		},
	}
}

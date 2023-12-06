package explorer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

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

func NewExplorer(policyFile string, tracesFile string) (*Explorer, error) {
	e := &Explorer{
		PolicyFile: policyFile,
		TracesFile: tracesFile,
		QTable:     policies.NewQTable(),
		Traces:     make([]*Trace, 0),
		StateMap:   make(map[string]*State),
	}

	if err := e.QTable.Read(policyFile); err != nil {
		return nil, err
	}
	if err := e.readTraces(); err != nil {
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

func (e *Explorer) readTraces() error {
	file, err := os.Open(e.TracesFile)
	if err != nil {
		return fmt.Errorf("error reading file: %s", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)
	for scanner.Scan() {
		t := NewTrace()
		err := json.Unmarshal(scanner.Bytes(), t)
		if err != nil {
			return fmt.Errorf("error reading file contents: %s", err)
		}
		e.Traces = append(e.Traces, t)
	}
	return nil
}

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

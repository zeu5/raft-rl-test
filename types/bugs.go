package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
)

type BugDesc struct {
	Name  string
	Check func(*Trace) (bool, int)
}

// checks all the bugs on the traces
type BugAnalyzer struct {
	Bugs        []BugDesc
	occurrences map[string][]int
	savePath    string
}

func NewBugAnalyzer(savePath string, bugs ...BugDesc) *BugAnalyzer {
	if _, ok := os.Stat(savePath); ok != nil {
		os.MkdirAll(savePath, 0777)
	}
	return &BugAnalyzer{
		Bugs:        bugs,
		occurrences: make(map[string][]int),
		savePath:    savePath,
	}
}

func BugAnalyzerCtor(savePath string, bugs ...BugDesc) func() Analyzer {
	return func() Analyzer {
		return NewBugAnalyzer(savePath, bugs...)
	}
}

func (ba *BugAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *Trace) {
	for _, b := range ba.Bugs {
		_, ok := ba.occurrences[b.Name]
		bugFound, step := b.Check(trace)
		if bugFound { // swapped order just to debug
			if !ok {
				ba.occurrences[b.Name] = make([]int, 0)
			}
			ba.occurrences[b.Name] = append(ba.occurrences[b.Name], startingTimestep+step)
			bugPath := path.Join(ba.savePath, strconv.Itoa(run)+"_"+s+"_"+b.Name+"_"+strconv.Itoa(episode)+"_step"+strconv.Itoa(step)+"_bug.json")
			trace.Record(bugPath)
		}
	}
}

func (ba *BugAnalyzer) DataSet() DataSet {
	out := make(map[string][]int)
	for b, i := range ba.occurrences {
		out[b] = make([]int, len(i))
		copy(out[b], i)
	}
	return out
}

func (ba *BugAnalyzer) Reset() {
	ba.occurrences = make(map[string][]int)
}

var _ Analyzer = (*BugAnalyzer)(nil)

func BugComparator(savePath string) Comparator {
	return func(run, _ int, expNames []string, ds map[string]DataSet) {
		data := make(map[string]map[string][]int)
		for _, exp := range expNames {
			bugOccurrences := ds[exp].(map[string][]int)
			fmt.Printf("For run:%d, experiment: %s\n", run, exp)
			for b, i := range bugOccurrences {
				fmt.Printf("\tBug: %s, First iteration: %d\n", b, i)
			}
			data[exp] = bugOccurrences
		}

		bs, err := json.Marshal(data)
		if err == nil {
			os.WriteFile(path.Join(savePath, strconv.Itoa(run)+"_bug.json"), bs, 0644)
		}
	}
}

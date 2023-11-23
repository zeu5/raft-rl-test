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
func BugAnalyzer(savePath string, bugs ...BugDesc) Analyzer {
	if _, ok := os.Stat(savePath); ok == nil {
		os.RemoveAll(savePath)
	}
	os.MkdirAll(savePath, 0777)
	return func(run int, s string, traces []*Trace) DataSet {
		firstOccurrence := make(map[string]int)
		for i, t := range traces {
			for _, b := range bugs {
				_, ok := firstOccurrence[b.Name]
				bugFound, step := b.Check(t)
				if bugFound { // swapped order just to debug
					if !ok {
						firstOccurrence[b.Name] = i
					}
					bugPath := path.Join(savePath, strconv.Itoa(run)+"_"+s+"_"+b.Name+"_"+strconv.Itoa(i)+"_step"+strconv.Itoa(step)+"_bug.json")
					t.Record(bugPath)
				}
			}
		}
		return firstOccurrence
	}
}

func BugComparator(savePath string) Comparator {
	return func(run int, s []string, ds []DataSet) {
		data := make(map[string]map[string]int)
		for i, exp := range s {
			bugOccurrences := ds[i].(map[string]int)
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

package cbft

import "github.com/zeu5/raft-rl-test/types"

func CoverageAnalyzer(colors ...CometColorFunc) types.Analyzer {
	return func(i int, s string, t []*types.Trace) types.DataSet {
		return nil
	}
}

func CoverageComparator(savePath string) types.Comparator {
	return func(i int, s []string, ds []types.DataSet) {}
}

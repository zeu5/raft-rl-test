package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

type PureCoverageAnalyzer struct {
	uniqueStates    map[string]bool
	numUniqueStates []int
}

func NewPureCoverageAnalyzer() *PureCoverageAnalyzer {
	return &PureCoverageAnalyzer{
		uniqueStates:    make(map[string]bool),
		numUniqueStates: make([]int, 0),
	}
}

func PureCoverageAnalyzerCtor() func() Analyzer {
	return func() Analyzer {
		return NewPureCoverageAnalyzer()
	}
}

func (pca *PureCoverageAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *Trace) {
	for j := 0; j < trace.Len(); j++ {
		s, _, _, _ := trace.Get(j)
		sHash := s.Hash()
		if _, ok := pca.uniqueStates[sHash]; !ok {
			pca.uniqueStates[sHash] = true
		}
	}
	pca.numUniqueStates = append(pca.numUniqueStates, len(pca.uniqueStates))
}

func (pca *PureCoverageAnalyzer) DataSet() DataSet {
	out := make([]int, len(pca.numUniqueStates))
	copy(out, pca.numUniqueStates)
	return out
}

func (pca *PureCoverageAnalyzer) Reset() {
	pca.uniqueStates = make(map[string]bool)
	pca.numUniqueStates = make([]int, 0)
}

var _ Analyzer = (*PureCoverageAnalyzer)(nil)

func PureCoveragePlotter(plotPath string) Comparator {
	if _, err := os.Stat(plotPath); err != nil {
		os.Mkdir(plotPath, os.ModePerm)
	}
	return func(i, _ int, s []string, ds []DataSet) {
		p := plot.New()
		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"
		for i := 0; i < len(s); i++ {
			uniqueStates := ds[i].([]int)
			points := make(plotter.XYs, len(uniqueStates))
			for i, v := range uniqueStates {
				points[i] = plotter.XY{
					X: float64(i),
					Y: float64(v),
				}
			}
			line, err := plotter.NewLine(points)
			if err != nil {
				continue
			}
			line.Color = plotutil.Color(i)
			p.Add(line)
			p.Legend.Add(s[i], line)
			fmt.Printf("Number of unique states: %d for benchmark: %s\n", uniqueStates[len(uniqueStates)-1], s[i])
		}
		p.Save(8*vg.Inch, 8*vg.Inch, path.Join(plotPath, strconv.Itoa(i)+"_pure_coverage.png"))
	}
}

type PartitionCoverage struct {
	uniqueStates    map[string]bool
	numUniqueStates []int
}

func NewPartitionCoverage() *PartitionCoverage {
	return &PartitionCoverage{
		uniqueStates:    make(map[string]bool),
		numUniqueStates: make([]int, 0),
	}
}

func (pc *PartitionCoverage) Analyze(run int, episode int, startingTimestep int, s string, trace *Trace) {
	for j := 0; j < trace.Len(); j++ {
		s, _, _, _ := trace.Get(j)
		sHash := partitionHash(s.(*Partition))
		if _, ok := pc.uniqueStates[sHash]; !ok {
			pc.uniqueStates[sHash] = true
		}
	}
	pc.numUniqueStates = append(pc.numUniqueStates, len(pc.uniqueStates))
}

func (pc *PartitionCoverage) DataSet() DataSet {
	out := make([]int, len(pc.numUniqueStates))
	copy(out, pc.numUniqueStates)
	return out
}

func (pc *PartitionCoverage) Reset() {
	pc.uniqueStates = make(map[string]bool)
	pc.numUniqueStates = make([]int, 0)
}

var _ Analyzer = (*PartitionCoverage)(nil)

func partitionHash(p *Partition) string {
	partition := make([][]string, len(p.Partition))
	for i, par := range p.Partition {
		partition[i] = make([]string, len(par))
		for j, color := range par {
			partition[i][j] = color.Hash()
		}
	}
	activeColors := make(map[string]bool)
	for node, c := range p.ReplicaColors {
		if _, ok := p.ActiveNodes[node]; ok {
			activeColors[c.Hash()] = true
		}
	}

	bs, _ := json.Marshal(map[string]interface{}{
		"colors":           partition,
		"repeat_count":     p.RepeatCount,
		"pending_requests": len(p.PendingRequests),
		"active_colors":    activeColors,
	})
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func PartitionCoveragePlotter(plotPath string) Comparator {
	if _, err := os.Stat(plotPath); err != nil {
		os.Mkdir(plotPath, os.ModePerm)
	}
	return func(i, _ int, s []string, ds []DataSet) {
		p := plot.New()
		p.Title.Text = "Comparison"
		p.X.Label.Text = "Iteration"
		p.Y.Label.Text = "States covered"
		for i := 0; i < len(s); i++ {
			uniqueStates := ds[i].([]int)
			points := make(plotter.XYs, len(uniqueStates))
			for i, v := range uniqueStates {
				points[i] = plotter.XY{
					X: float64(i),
					Y: float64(v),
				}
			}
			line, err := plotter.NewLine(points)
			if err != nil {
				continue
			}
			line.Color = plotutil.Color(i)
			p.Add(line)
			p.Legend.Add(s[i], line)
			fmt.Printf("Number of unique states: %d for benchmark: %s\n", uniqueStates[len(uniqueStates)-1], s[i])
		}
		p.Save(8*vg.Inch, 8*vg.Inch, path.Join(plotPath, strconv.Itoa(i)+"_partition_coverage.png"))
	}
}

type CrashAnalyzer struct {
	numCrashes []int
}

func NewCrashAnalyzer() *CrashAnalyzer {
	return &CrashAnalyzer{
		numCrashes: make([]int, 0),
	}
}

func CrashAnalyzerCtor() func() Analyzer {
	return func() Analyzer {
		return NewCrashAnalyzer()
	}
}

func (ca *CrashAnalyzer) Analyze(run int, episode int, startingTimestep int, s string, trace *Trace) {
	crashes := 0
	for j := 0; j < trace.Len(); j++ {
		_, a, _, _ := trace.Get(j)
		if ss, ok := a.(*StopStartAction); ok && ss.Action == "Stop" {
			crashes += 1
		}
	}
	ca.numCrashes = append(ca.numCrashes, crashes)
}

func (ca *CrashAnalyzer) DataSet() DataSet {
	out := make([]int, len(ca.numCrashes))
	copy(out, ca.numCrashes)
	return out
}

func (ca *CrashAnalyzer) Reset() {
	ca.numCrashes = make([]int, 0)
}

var _ Analyzer = (*CrashAnalyzer)(nil)

func CrashComparator(saveFile string) Comparator {
	if _, err := os.Stat(saveFile); err != nil {
		os.Mkdir(saveFile, os.ModePerm)
	}
	return func(run, _ int, names []string, datasets []DataSet) {
		for i := 0; i < len(names); i++ {
			p := plot.New()
			p.Title.Text = "Comparison"
			p.X.Label.Text = "Iteration"
			p.Y.Label.Text = "States covered"
			crashDataSet := datasets[i].([]int)
			points := make(plotter.XYs, len(crashDataSet))
			for i, v := range crashDataSet {
				points[i] = plotter.XY{
					X: float64(i),
					Y: float64(v),
				}
			}
			line, err := plotter.NewLine(points)
			if err != nil {
				continue
			}
			line.Color = plotutil.Color(i)
			p.Add(line)
			p.Legend.Add(names[i], line)
			p.Save(8*vg.Inch, 8*vg.Inch, path.Join(saveFile, strconv.Itoa(run)+"_"+names[i]+"_crashes.png"))
		}
	}
}

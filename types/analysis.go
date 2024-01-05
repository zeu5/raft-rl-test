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

func PureCoverage() Analyzer {
	return func(i int, s string, t []*Trace) DataSet {
		uniqueStates := make(map[string]bool)
		numUniqueStates := make([]int, 0)
		for _, trace := range t {
			for j := 0; j < trace.Len(); j++ {
				s, _, _, _ := trace.Get(j)
				sHash := s.Hash()
				if _, ok := uniqueStates[sHash]; !ok {
					uniqueStates[sHash] = true
				}
			}
			numUniqueStates = append(numUniqueStates, len(uniqueStates))
		}
		return numUniqueStates
	}
}

func PureCoveragePlotter(plotPath string) Comparator {
	if _, err := os.Stat(plotPath); err != nil {
		os.Mkdir(plotPath, os.ModePerm)
	}
	return func(i int, s []string, ds []DataSet) {
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

func PartitionCoverage() Analyzer {
	return func(i int, s string, t []*Trace) DataSet {
		uniqueStates := make(map[string]bool)
		numUniqueStates := make([]int, 0)
		for _, trace := range t {
			for j := 0; j < trace.Len(); j++ {
				s, _, _, _ := trace.Get(j)
				sHash := partitionHash(s.(*Partition))
				if _, ok := uniqueStates[sHash]; !ok {
					uniqueStates[sHash] = true
				}
			}
			numUniqueStates = append(numUniqueStates, len(uniqueStates))
		}
		return numUniqueStates
	}
}

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
	return func(i int, s []string, ds []DataSet) {
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

func CrashAnalyzer() Analyzer {
	return func(i int, s string, t []*Trace) DataSet {
		numCrashes := make([]int, len(t))
		for i, trace := range t {
			crashes := 0
			for j := 0; j < trace.Len(); j++ {
				_, a, _, _ := trace.Get(j)
				if ss, ok := a.(*StopStartAction); ok && ss.Action == "Stop" {
					crashes += 1
				}
			}
			numCrashes[i] = crashes
		}
		return numCrashes
	}
}

func CrashComparator(saveFile string) Comparator {
	if _, err := os.Stat(saveFile); err != nil {
		os.Mkdir(saveFile, os.ModePerm)
	}
	return func(run int, names []string, datasets []DataSet) {
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

// takes a save path and a variable number of strings and writes them to file separated by new lines
func WriteToFile(savePath string, content ...string) {
	singleString := ""
	for _, c := range content {
		singleString = fmt.Sprintf("%s \n%s", singleString, c)
	}

	os.WriteFile(savePath, []byte(singleString), 0644)
}

func AppendToFile(savePath string, content ...string) {
	f, err := os.OpenFile(savePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	for _, s := range content {
		if _, err = f.WriteString(s + "\n"); err != nil {
			panic(err)
		}
	}
}

func RecordPartitionStats(savePath string) EAnalyzer {
	return func(run int, e *Experiment) {
		statsSavePath := path.Join(savePath, strconv.Itoa(run)+"_"+e.Name+"_stats.json")
		if partEnv, ok := e.Environment.(*PartitionEnv); ok {
			partEnv.RecordStats(statsSavePath)
			partEnv.ResetStats()
		}
	}
}

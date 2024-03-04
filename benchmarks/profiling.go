package benchmarks

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
)

func startProfiling() {
	if cpuprofile != "" {
		// cpuProfPath := saveFile + "_" + cpuprofile
		cpuProfPath := path.Join(saveFile, cpuprofile)
		fmt.Println("Profiling CPU to ", cpuProfPath)
		f, err := os.Create(cpuProfPath)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	if memprofile != "" {
		// memProfPath := saveFile + "_" + memprofile
		memProfPath := path.Join(saveFile, memprofile)
		fmt.Println("Profiling Memory to ", memProfPath)
		f, err := os.Create(memProfPath)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}

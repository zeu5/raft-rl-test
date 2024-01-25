package types

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/zeu5/raft-rl-test/util"
)

type EpisodeContext struct {
	// parameters used to run episode
	Context        context.Context
	Cancel         context.CancelFunc // cancel function to stop the episode
	Episode        int
	Step           int
	ExperimentName string

	reportSavePath    string
	reportPrintConfig *ReportsPrintConfig

	// parameters to be used post running episode
	Err         error          // error of the episode
	TimedOut    bool           // true if the episode timed out
	Trace       *Trace         // trace of the episode
	Report      *EpisodeReport // report of the episode
	RunDuration time.Duration  // duration of the episode
}

// Context related with each step
type StepContext struct {
	// Wraps the episode context
	*EpisodeContext
	// State of this current state
	State State
	// Action taken at this step
	Action Action
	// Next state at this step
	NextState State
	// Additional info related to step
	addInfo map[string]interface{}
}

// Create a new step context with the episode context
func NewStepContext(eCtx *EpisodeContext) *StepContext {
	return &StepContext{
		EpisodeContext: eCtx,
		addInfo:        make(map[string]interface{}),
	}
}

// Add info to this step
func (s *StepContext) AddInfo(key string, value interface{}) {
	s.addInfo[key] = value
}

// Create a new episode context
func NewEpisodeContext(episodeNumber int, experimentName string, eConfig *experimentRunConfig) *EpisodeContext {
	e := &EpisodeContext{
		Episode:           episodeNumber,
		ExperimentName:    experimentName,
		Report:            NewEpisodeReport(episodeNumber, experimentName),
		Trace:             NewTrace(),
		reportSavePath:    eConfig.ReportSavePath,
		reportPrintConfig: eConfig.ReportsPrintConfig,
	}
	if eConfig.Timeout > 0 {
		toCtx, toCancel := context.WithTimeout(eConfig.Context, eConfig.Timeout)
		e.Context = toCtx
		e.Cancel = toCancel
	} else {
		e.Context, e.Cancel = context.WithCancel(eConfig.Context)
	}
	return e
}

// Update the step and also in the report
func (e *EpisodeContext) SetStep(step int) {
	e.Step = step
	e.Report.setEpisodeStep(step)
}

// Set the error
func (e *EpisodeContext) SetError(err error) {
	e.Err = err
}

// Set timeout flag to true
func (e *EpisodeContext) SetTimedOut() {
	e.TimedOut = true
}

// Record to the report to the path "reportSavePath" using "reportPrintConfig"
func (e *EpisodeContext) RecordReport() {
	// TODO: complete this function
	reason := ""
	if e.TimedOut {
		reason = "TIMEOUT\n" + fmt.Sprintf("timeout: %s", e.RunDuration.String())
		if e.Err != nil {
			reason = fmt.Sprintf("%s\nerror: %s", reason, e.Err.Error())
		}
	} else if e.Err != nil {
		reason = "ERROR\n" + fmt.Sprintf("error: %s", e.Err.Error())
	} else {
		reason = "randomly sampled"
	}

	if e.reportPrintConfig.PrintStd {
		e.Report.Store(e.reportSavePath, reason)
	}
	if e.reportPrintConfig.PrintValues {
		e.Report.StoreValues(e.reportSavePath, reason)
	}
	if e.reportPrintConfig.PrintTimeline {
		e.Report.StoreTimeline(e.reportSavePath, reason)
	}
}

// REPORT CONFIGURATION

// Configuration of the report
type ReportsPrintConfig struct {
	PrintStd      bool // print the report standard representation
	PrintValues   bool // print the report values representation
	PrintTimeline bool // print the report timeline representation

	PrintIfError   bool // print the report if an error occurs
	PrintIfTimeout bool // print the report if a timeout occurs

	Sampling float32 // rate of randomly printed reports (for successful episodes)
}

// configuration of the report with no printing
func RepConfigOff() *ReportsPrintConfig {
	return &ReportsPrintConfig{
		PrintStd:      false,
		PrintValues:   false,
		PrintTimeline: false,

		PrintIfError:   false,
		PrintIfTimeout: false,

		Sampling: 0.0,
	}
}

// configuration of the report with standard printing, prints std and values version for both errors and timeouts. Prints a successfull episode with probability 0.02
func RepConfigStandard() *ReportsPrintConfig {
	return &ReportsPrintConfig{
		PrintStd:      true,
		PrintValues:   true,
		PrintTimeline: false,

		PrintIfError:   true,
		PrintIfTimeout: true,

		Sampling: 0.02,
	}
}

// configuration of the report with complete printing, prints std, values and timeline version for both errors and timeouts. Prints a successfull episode with probability 0.05
func RepConfigComplete() *ReportsPrintConfig {
	return &ReportsPrintConfig{
		PrintStd:      true,
		PrintValues:   true,
		PrintTimeline: true,

		PrintIfError:   true,
		PrintIfTimeout: true,

		Sampling: 1.0,
	}
}

// EPISODE REPORT

// Report of an episode
type EpisodeReport struct {
	EpisodeNumber  int
	ExperimentName string
	episodeStep    int

	nextIndex int       // next available index for an entry
	startTime time.Time // start time to compute timestamp of an entry

	lock *sync.Mutex // mutex to control entries updates

	Timeline   []*EpisodeReportEntry // generic timeline containing all the entries ordered by index
	TimeValues map[string][]*EpisodeReportEntry
	IntValues  map[string][]*EpisodeReportEntry
	Logs       map[string]string
}

func NewEpisodeReport(episodeNumber int, experimentName string) *EpisodeReport {
	return &EpisodeReport{
		EpisodeNumber:  episodeNumber,
		ExperimentName: experimentName,
		episodeStep:    0,

		nextIndex: 0,
		startTime: time.Now(),

		lock: &sync.Mutex{},

		Timeline:   make([]*EpisodeReportEntry, 0),
		TimeValues: make(map[string][]*EpisodeReportEntry),
		IntValues:  make(map[string][]*EpisodeReportEntry),
		Logs:       make(map[string]string),
	}
}

// set the current episode step in the report
func (e *EpisodeReport) setEpisodeStep(step int) {
	e.episodeStep = step
}

// add a new entry of type int to the report
func (e *EpisodeReport) AddIntEntry(value int, entryType string, caller string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	entry := EpisodeReportEntry{
		Index:     e.nextIndex,
		Timestamp: time.Since(e.startTime),

		EpisodeStep: e.episodeStep,
		EntryType:   entryType,
		Caller:      caller,
		Value:       value,
	}

	e.nextIndex += 1
	e.Timeline = append(e.Timeline, &entry)

	values, ok := e.IntValues[entryType]
	if !ok {
		values = make([]*EpisodeReportEntry, 0)
	}
	values = append(values, &entry)
	e.IntValues[entryType] = values
}

// add a new entry of type time.Duration to the report
func (e *EpisodeReport) AddTimeEntry(value time.Duration, entryType string, caller string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	entry := EpisodeReportEntry{
		Index:     e.nextIndex,
		Timestamp: time.Since(e.startTime),

		EpisodeStep: e.episodeStep,
		EntryType:   entryType,
		Caller:      caller,
		Value:       value,
	}

	e.nextIndex += 1
	e.Timeline = append(e.Timeline, &entry)

	values, ok := e.TimeValues[entryType]
	if !ok {
		values = make([]*EpisodeReportEntry, 0)
	}
	values = append(values, &entry)
	e.TimeValues[entryType] = values
}

func (e *EpisodeReport) AddLog(value string, key string) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.Logs[key] = value
}

// return a string representation of the report timeline
func (e *EpisodeReport) StringTimeline() string {
	result := "Length: " + fmt.Sprintf("%d", len(e.Timeline)) + "\n"
	result = fmt.Sprintf("%s%s", result, StringEntriesList(e.Timeline))
	return result
}

// return a string representation of the report entries per type
func (e *EpisodeReport) StringPerType() string {
	result := ""
	for entryType, entries := range e.TimeValues {
		result = fmt.Sprintf("%s\n%s [%d]:\n%s", result, entryType, len(entries), StringEntriesListLite(entries))
	}
	for entryType, entries := range e.IntValues {
		result = fmt.Sprintf("%s\n%s [%d]:\n%s", result, entryType, len(entries), StringEntriesListLite(entries))
	}
	for key, value := range e.Logs {
		result = fmt.Sprintf("%s\n%s :\n%s", result, key, value)
	}
	return result
}

// return a string representation of the report entries values per type
func (e *EpisodeReport) StringPerTypeValues() string {
	result := ""
	for entryType, entries := range e.TimeValues {
		result = fmt.Sprintf("%s\n%s :\n%s\n", result, entryType, StringEntriesValuesList(entries))
	}
	for entryType, entries := range e.IntValues {
		result = fmt.Sprintf("%s\n%s :\n%s\n", result, entryType, StringEntriesValuesList(entries))
	}
	return result
}

// return a string representation of the report entries of the specified type
func (e *EpisodeReport) StringSingleType(entryType string) string {
	if entries, ok := e.TimeValues[entryType]; ok {
		return fmt.Sprintf("%s :\n%s", entryType, StringEntriesList(entries))
	}
	if entries, ok := e.IntValues[entryType]; ok {
		return fmt.Sprintf("%s :\n%s", entryType, StringEntriesList(entries))
	}
	return "unknown entry type: " + entryType
}

// return a string representation of the report entries values of the specified type
func (e *EpisodeReport) StringSingleTypeValues(entryType string) string {
	if entries, ok := e.TimeValues[entryType]; ok {
		return fmt.Sprintf("%s :\n%s", entryType, StringEntriesValuesList(entries))
	}
	if entries, ok := e.IntValues[entryType]; ok {
		return fmt.Sprintf("%s :\n%s", entryType, StringEntriesValuesList(entries))
	}
	return "unknown entry type: " + entryType
}

// store the report in the specified path with the specified reason and type
func (e *EpisodeReport) store(basePath string, reason string, t string) {

	result := fmt.Sprintf("Exp: %s - Episode: %d\n\n%s\n", e.ExperimentName, e.EpisodeNumber, reason)
	result = fmt.Sprintf("%s\n", result)

	data := ""
	fileName := "std"
	switch t {
	case "std":
		data = e.StringPerType()
	case "values":
		data = e.StringPerTypeValues()
		fileName = "values"
	case "timeline":
		data = e.StringTimeline()
		fileName = "timeline"
	}

	filePath := path.Join(basePath, "epReports", fmt.Sprintf("%s_%d_report_"+fileName+".txt", e.ExperimentName, e.EpisodeNumber))
	result = fmt.Sprintf("%s%s\n", result, data)
	util.WriteToFile(filePath, result)
}

// store the report in the specified path
func (e *EpisodeReport) Store(basePath string, reason string) {
	e.store(basePath, reason, "std")
}

// store the report values in the specified path
func (e *EpisodeReport) StoreValues(basePath string, reason string) {
	e.store(basePath, reason, "values")
}

// store the report timeline in the specified path
func (e *EpisodeReport) StoreTimeline(basePath string, reason string) {
	e.store(basePath, reason, "timeline")
}

// ENTRY

// Entry of the Report
type EpisodeReportEntry struct {
	Index     int           // index of the entry, managed by the report
	Timestamp time.Duration // timestamp of the entry, managed by the report

	EpisodeStep int         // episode step
	EntryType   string      // entry type
	Caller      string      // the method adding the entry
	Value       interface{} // entry value
}

// return a string representation of the entry
func (en *EpisodeReportEntry) String() string {
	switch en.Value.(type) {
	case time.Duration:
		return fmt.Sprintf("[ %6d | %5d | %3d ] %20s : %12s (%20s)", en.Index, en.Timestamp.Milliseconds(), en.EpisodeStep, en.EntryType, en.Value.(time.Duration).String(), en.Caller)
	case int:
		return fmt.Sprintf("[ %6d | %5d | %3d ] %20s : %5d (%20s)", en.Index, en.Timestamp.Milliseconds(), en.EpisodeStep, en.EntryType, en.Value.(int), en.Caller)
	default:
		return "wrong entry type"
	}
}

// return a string representation of the entry value
func (en *EpisodeReportEntry) StringValue() string {
	switch en.Value.(type) {
	case time.Duration:
		return fmt.Sprintf("%d", en.Value.(time.Duration).Milliseconds())
	case int:
		return fmt.Sprintf("%3d", en.Value.(int))
	}
	return "N/A"
}

func (en *EpisodeReportEntry) StringLite() string {
	switch en.Value.(type) {
	case time.Duration:
		return fmt.Sprintf("[ %5d | %3d ] %12s (%20s)", en.Timestamp.Milliseconds(), en.EpisodeStep, en.Value.(time.Duration).String(), en.Caller)
	case int:
		return fmt.Sprintf("[ %5d | %3d ] %5d (%20s)", en.Timestamp.Milliseconds(), en.EpisodeStep, en.Value.(int), en.Caller)
	default:
		return "wrong entry type"
	}
}

// return a string representation of the list of entries
func StringEntriesList(list []*EpisodeReportEntry) string {
	result := ""
	for _, entry := range list {
		result = fmt.Sprintf("%s%s\n", result, entry.String())
	}
	return result
}

// return a string representation of the list of entries
func StringEntriesListLite(list []*EpisodeReportEntry) string {
	result := ""
	for _, entry := range list {
		result = fmt.Sprintf("%s%s\n", result, entry.StringLite())
	}
	return result
}

// return a string representation of the list of entries values
func StringEntriesValuesList(list []*EpisodeReportEntry) string {
	result := ""
	for i, entry := range list {
		result = fmt.Sprintf("%s %s", result, entry.StringValue())
		if (i+1)%20 == 0 {
			result = fmt.Sprintf("%s\n", result)
		}
	}
	return result
}

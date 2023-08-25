package rsl

type Entry struct {
	Accepted bool
	Ballot   Ballot
	Decree   int
	Command  Command
}

type Log struct {
	entries []Entry
	decided []Command
}

func NewLog() *Log {
	return &Log{
		entries: make([]Entry, 0),
		decided: make([]Command, 0),
	}
}

func (l *Log) Add(e Entry) {
	l.entries = append(l.entries, e)
}

func (l *Log) AddDecided(c Command) {
	l.decided = append(l.decided, c.Copy())
}

func (l *Log) NumDecided() int {
	return len(l.decided)
}

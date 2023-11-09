package rsl

type Entry struct {
	Accepted bool
	Ballot   Ballot
	Decree   int
	Command  Command
}

func (e Entry) Copy() Entry {
	return Entry{
		Accepted: e.Accepted,
		Ballot:   e.Ballot.Copy(),
		Decree:   e.Decree,
		Command:  e.Command.Copy(),
	}
}

type Log struct {
	entries []Entry
	Decided []Command
}

func NewLog() *Log {
	return &Log{
		entries: make([]Entry, 0),
		Decided: make([]Command, 0),
	}
}

func (l *Log) Add(e Entry) {
	l.entries = append(l.entries, e)
}

func (l *Log) AddDecided(c Command) {
	l.Decided = append(l.Decided, c.Copy())
}

func (l *Log) NumDecided() int {
	return len(l.Decided)
}

func (l *Log) Copy() *Log {
	out := NewLog()
	for _, e := range l.entries {
		out.Add(e.Copy())
	}
	for _, c := range l.Decided {
		out.AddDecided(c.Copy())
	}
	return out
}

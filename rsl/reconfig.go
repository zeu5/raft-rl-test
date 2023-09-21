package rsl

import (
	"bytes"
	"encoding/json"
)

// Configuration of a RSL node, contains the set of peers
type RSLConfig struct {
	Number        int
	InitialDecree int
	Members       map[uint64]bool
}

func (r RSLConfig) Copy() RSLConfig {
	n := RSLConfig{
		Number:        r.Number,
		InitialDecree: r.InitialDecree,
		Members:       make(map[uint64]bool),
	}
	for k := range r.Members {
		n.Members[k] = true
	}
	return n
}

type RSLConfigCommand struct {
	Members map[uint64]bool
}

func NewRSLConfigCommand(peers []uint64) *RSLConfigCommand {
	c := &RSLConfigCommand{
		Members: make(map[uint64]bool),
	}
	for _, p := range peers {
		c.Members[p] = true
	}
	return c
}

func (r *RSLConfigCommand) Add(peer uint64) {
	r.Members[peer] = true
}

func (r *RSLConfigCommand) Remove(peer uint64) {
	delete(r.Members, peer)
}

func (r *RSLConfigCommand) ToCommand() Command {
	c := Command{}
	bs, _ := json.Marshal(r)
	c.Data = bytes.Clone(bs)
	return c
}

func IsConfigCommand(c Command) bool {
	cfg := RSLConfigCommand{}
	err := json.Unmarshal(c.Data, &cfg)
	return err == nil
}

func GetRSLConfig(c Command) (*RSLConfigCommand, bool) {
	cfg := &RSLConfigCommand{}
	err := json.Unmarshal(c.Data, cfg)
	if err != nil {
		return nil, false
	}
	return cfg, true
}

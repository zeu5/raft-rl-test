package util

import (
	"sort"
	"strconv"
)

type Elem interface {
	Key() string
}

type StringElem string

func (s StringElem) Key() string {
	return string(s)
}

type IntElem int

func (i IntElem) Key() string {
	return strconv.Itoa(int(i))
}

type MultiSet []Elem

func (m MultiSet) toMap() map[string][]Elem {
	result := make(map[string][]Elem)
	for _, e := range m {
		key := e.Key()
		if _, ok := result[key]; !ok {
			result[key] = make([]Elem, 0)
		}
		result[key] = append(result[key], e)
	}
	return result
}

func (m MultiSet) keysNMultiplicities() ([]string, []int) {
	sMap := m.toMap()
	sortedKeys := make([]string, 0)
	for k := range sMap {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	multiplicities := make([]int, len(sortedKeys))
	for i, sKey := range sortedKeys {
		multiplicities[i] = len(sMap[sKey])
	}
	return sortedKeys, multiplicities
}

type Partition [][]Elem

func (p Partition) Eq(other Partition) bool {
	if len(p) != len(other) {
		return false
	}
	for i := 0; i < len(p); i++ {
		if len(p[i]) != len(other[i]) {
			return false
		}
		for j := 0; j < len(p[i]); j++ {
			if p[i][j] != other[i][j] {
				return false
			}
		}
	}
	return true
}

type stack struct {
	c int
	u int
	v int
}

type visitorFunc func([][]string)

func enumeratePartitions(multiplicities []int, keys []string, visitor visitorFunc) {
	m := len(multiplicities)
	n := 0
	for _, count := range multiplicities {
		n += count
	}

	// M1. Initialize
	stacks := make([]stack, m*n+1)
	for i := 0; i < m*n+1; i++ {
		if i < m {
			stacks[i] = stack{
				c: i + 1,
				u: multiplicities[i],
				v: multiplicities[i],
			}
		} else {
			stacks[i] = stack{}
		}
	}
	f := make([]int, n+1)
	f[0] = 0
	a := 0
	l := 0
	f[1] = m
	b := m

	for {
		var j int
		for {
			// M2. Subtract u from v
			j = a
			k := b
			x := false
			for j < b {
				stacks[k].u = stacks[j].u - stacks[j].v
				if stacks[k].u == 0 {
					x = true
				} else if !x {
					stacks[k].c = stacks[j].c
					stacks[k].v = stacks[j].v
					if stacks[k].u < stacks[j].v {
						stacks[k].v = stacks[k].u
						x = true
					}
					k = k + 1
				} else {
					stacks[k].c = stacks[j].c
					stacks[k].v = stacks[k].u
					k = k + 1
				}
				j = j + 1
			}

			// M3. Push if non zero
			if k > b {
				a = b
				b = k
				l = l + 1
				f[l+1] = b
				// Return to M2
			} else {
				break
			}
		}

		// M4. Visit a partition
		parts := make([][]string, 0)
		for p := 0; p < l+1; p++ {
			part := make([]string, 0)
			for q := f[p]; q < f[p+1]; q++ {
				key := keys[stacks[q].c-1]
				for r := 0; r < stacks[q].v; r++ {
					part = append(part, key)
				}
			}
			parts = append(parts, part)
		}
		visitor(parts)

		for {
			// M5. Decrease v
			j = b - 1
			for stacks[j].v == 0 {
				j = j - 1
			}
			if j == a && stacks[j].v == 1 {
				// M6. Backtrack
				if l == 0 {
					return
				}
				l = l - 1
				b = a
				a = f[l]
				// Return to M5
			} else {
				stacks[j].v = stacks[j].v - 1
				for k := j + 1; k < b; k++ {
					stacks[k].v = stacks[k].u
				}
				// Go back to M2
				break
			}
		}
	}
}

// Enumerate Partitions lists all the possible partitions
// of a multiset
// The algorithm is drawn from "The Art of Computer Programming"
// Section 7.2.1.5, Algorithm M

func EnumeratePartitions(s MultiSet) []Partition {
	partitions := make([]Partition, 0)
	sMap := s.toMap()
	partitionVisitor := func(partition [][]string) {
		ePartition := make([][]Elem, 0)
		indices := make(map[string]int)
		for k := range sMap {
			indices[k] = 0
		}
		for _, part := range partition {
			ePart := make([]Elem, 0)
			for _, k := range part {
				i := indices[k]
				elems := sMap[k]
				ePart = append(ePart, elems[i])
				indices[k] = i + 1
			}
			ePartition = append(ePartition, ePart)
		}
		partitions = append(partitions, ePartition)
	}
	keys, multiplicities := s.keysNMultiplicities()
	enumeratePartitions(multiplicities, keys, partitionVisitor)
	return partitions
}

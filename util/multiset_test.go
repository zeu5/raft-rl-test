package util

import (
	"testing"
)

func TestEnumerateMultiSet(t *testing.T) {
	multiSet := []Elem{StringElem("a"), StringElem("b"), StringElem("b")}
	expected := []Partition{
		[][]Elem{{StringElem("a"), StringElem("b"), StringElem("b")}},
		[][]Elem{{StringElem("a"), StringElem("b")}, {StringElem("b")}},
		[][]Elem{{StringElem("a")}, {StringElem("b"), StringElem("b")}},
		[][]Elem{{StringElem("a")}, {StringElem("b")}, {StringElem("b")}},
	}
	obtained := EnumeratePartitions(multiSet)
	if len(obtained) != len(expected) {
		t.Errorf("incorrect number of partitions obtained")
	}
	for i := 0; i < len(expected); i++ {
		if !expected[i].Eq(obtained[i]) {
			t.Errorf("incorrect partition")
		}
	}
}

package levenshtein_test

import (
	"fmt"
	"testing"

	agnivade "github.com/agnivade/levenshtein"
	arbovm "github.com/arbovm/levenshtein"
	dgryski "github.com/dgryski/trifles/leven"
	kaweihe "github.com/ka-weihe/fast-levenshtein"	
)

func TestDistanceLib(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"a", "a", 0},
		{"ab", "ab", 0},
		{"ab", "aa", 1},
		{"ab", "aa", 1},
		{"ab", "aaa", 2},
		{"bbb", "a", 3},
		{"kitten", "sitting", 3},
		{"aa", "aü", 1},
		{"Fön", "Föm", 1},
	}
	for i, d := range tests {
		var n int
		n = agnivade.ComputeDistance(d.a, d.b)
		if n != d.want {
			t.Errorf("agnivade: Test[%d]: ComputeDistance(%q,%q) returned %v, want %v",
				i, d.a, d.b, n, d.want)
		}
		n = dgryski.Levenshtein([]rune(d.a), []rune(d.b))
		if n != d.want {
			t.Errorf("dgryski: Test[%d]: ComputeDistance(%q,%q) returned %v, want %v",
				i, d.a, d.b, n, d.want)
		}
		n = arbovm.Distance(d.a, d.b)
		if n != d.want {
			t.Errorf("arbovm: Test[%d]: ComputeDistance(%q,%q) returned %v, want %v",
				i, d.a, d.b, n, d.want)
		}
		n = kaweihe.Distance(d.a, d.b)
		if n != d.want {
			t.Errorf("arbovm: Test[%d]: ComputeDistance(%q,%q) returned %v, want %v",
				i, d.a, d.b, n, d.want)
		}	
	}
	fmt.Println("PASS arbovm")
}


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
		{"", "hello", 5},
		{"hello", "", 5},
		{"hello", "hello", 0},
		{"ab", "aa", 1},
		{"ab", "ba", 2},
		{"ab", "aaa", 2},
		{"bbb", "a", 3},
		{"kitten", "sitting", 3},
		{"distance", "difference", 5},
		{"levenshtein", "frankenstein", 6},
		{"resume and cafe", "resumes and cafes", 2},
		{"a very long string that is meant to exceed", "another very long string that is meant to exceed", 6},	
		{"resumé and café", "resumés and cafés", 2},
		{"resume and cafe", "resumé and café", 2},
		{"Hafþór Júlíus Björnsson", "Hafþor Julius Bjornsson", 4},
		{"།་གམ་འས་པ་་མ།", "།་གམའས་པ་་མ", 2},
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
	fmt.Println("PASS")
}

// // Benchmarks
// // ----------------------------------------------
// var sink int

// func BenchmarkAll(b *testing.B) {
// 	tests := []struct {
// 		a, b string
// 		name string
// 	}{
// 		// ASCII
// 		// {"levenshtein", "frankenstein", "ASCII"},
// 		// // Testing acutes and umlauts
// 		// {"resumé and café", "resumés and cafés", "French"},
// 		// {"Hafþór Júlíus Björnsson", "Hafþor Julius Bjornsson", "Nordic"},
// 		// // Only 2 characters are less in the 2nd string
// 		// {"།་གམ་འས་པ་་མ།", "།་གམའས་པ་་མ", "Tibetan"},
// 		{"batt den 1", "bật đèn 1", "VietNam"},
// 	}
// 	tmp := 0
// 	for _, test := range tests {
// 		b.Run(test.name, func(b *testing.B) {
// 			b.Run("agniva", func(b *testing.B) {
// 				for n := 0; n < b.N; n++ {
// 					tmp = agnivade.ComputeDistance(test.a, test.b)
// 				}
// 			})
// 			b.Run("arbovm", func(b *testing.B) {
// 				for n := 0; n < b.N; n++ {
// 					tmp = arbovm.Distance(test.a, test.b)
// 				}
// 			})
// 			b.Run("dgryski", func(b *testing.B) {
// 				for n := 0; n < b.N; n++ {
// 					tmp = dgryski.Levenshtein([]rune(test.a), []rune(test.b))
// 				}
// 			})
// 			b.Run("kaweihe", func(b *testing.B) {
// 				for n := 0; n < b.N; n++ {
// 					tmp = kaweihe.Distance(test.a, test.b)
// 				}
// 			})
// 		})
// 	}
// 	sink = tmp
// }

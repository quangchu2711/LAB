package levenshtein_test

import (
	"testing"

	agnivade "github.com/agnivade/levenshtein"
	arbovm "github.com/arbovm/levenshtein"
	dgryski "github.com/dgryski/trifles/leven"
	kaweihe "github.com/ka-weihe/fast-levenshtein"	
)


// Benchmarks
// ----------------------------------------------
var sink int

func BenchmarkAll(b *testing.B) {
	tests := []struct {
		a, b string
		name string
	}{
		// ASCII
		// {"levenshtein", "frankenstein", "ASCII"},
		// // Testing acutes and umlauts
		// {"resumé and café", "resumés and cafés", "French"},
		// {"Hafþór Júlíus Björnsson", "Hafþor Julius Bjornsson", "Nordic"},
		// // Only 2 characters are less in the 2nd string
		// {"།་གམ་འས་པ་་མ།", "།་གམའས་པ་་མ", "Tibetan"},
		{"batt den 1", "bật đèn 1", "VietNam"},
	}
	tmp := 0
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			b.Run("agniva", func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					tmp = agnivade.ComputeDistance(test.a, test.b)
				}
			})
			b.Run("arbovm", func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					tmp = arbovm.Distance(test.a, test.b)
				}
			})
			b.Run("dgryski", func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					tmp = dgryski.Levenshtein([]rune(test.a), []rune(test.b))
				}
			})
			b.Run("kaweihe", func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					tmp = kaweihe.Distance(test.a, test.b)
				}
			})
		})
	}
	sink = tmp
}

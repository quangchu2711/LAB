package hello

import "testing"

func BenchmarkRuneCount2(b *testing.B) {
    s := "Gophers are amazing 😁"
    // for i := 0; i < b.N; i++ {
        RuneCount2(s)
    // }
}

func BenchmarkRuneCount(b *testing.B) {
    s := "Gophers are amazing 😁"
    // for i := 0; i < b.N; i++ {
        RuneCount(s)
    // }
}

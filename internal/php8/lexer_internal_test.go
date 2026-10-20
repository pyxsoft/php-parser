package php8

import (
	"strconv"
	"strings"
	"testing"
)

func TestFitsInt64(t *testing.T) {
	cases := []struct {
		digits string
		base   uint64
	}{
		{"0", 10}, {"1", 10}, {"1_000_000", 10}, {"9223372036854775807", 10},
		{"9223372036854775808", 10}, {"99999999999999999999", 10},
		{"0777", 8}, {"08", 8}, {"0", 8},
		{"7fffffffffffffff", 16}, {"8000000000000000", 16}, {"DEAD_beef", 16},
		{"777777777777777777777", 8}, {"1777777777777777777777", 8},
		{"0101", 2}, {strings.Repeat("1", 63), 2}, {strings.Repeat("1", 64), 2},
		{"", 10}, {"_", 10}, {"12a", 10},
	}

	for _, c := range cases {
		_, err := strconv.ParseInt(strings.ReplaceAll(c.digits, "_", ""), int(c.base), 64)
		want := err == nil
		if got := fitsInt64([]byte(c.digits), c.base); got != want {
			t.Errorf("fitsInt64(%q, %d) = %v, want %v", c.digits, c.base, got, want)
		}
	}
}

func BenchmarkFitsInt64(b *testing.B) {
	d := []byte("1_234_567_890")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fitsInt64(d, 10)
	}
}

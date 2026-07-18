package service

import "testing"

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0.00"},
		{5, "5.00"},
		{1234.5, "1,234.50"},
		{120000, "120,000.00"},
		{1000000, "1,000,000.00"},
		{-2500.75, "-2,500.75"},
	}
	for _, c := range cases {
		if got := formatMoney(c.in); got != c.want {
			t.Errorf("formatMoney(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

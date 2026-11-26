package main

import (
	"math"
	"testing"
)

func TestFailureRate(t *testing.T) {
	cases := []struct {
		mode, rate string
		want       float64
		wantErr    bool
	}{
		{"", "", 0, false},
		{"false", "0.5", 0, false},
		{"true", "", 0.1, false},
		{"true", "0.3", 0.3, false},
		{"true", "2", 0, true},
		{"true", "x", 0, true},
	}
	for _, c := range cases {
		t.Setenv("FAILURE_MODE", c.mode)
		t.Setenv("FAILURE_RATE", c.rate)
		got, err := failureRate()
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("mode=%q rate=%q: got %v, %v", c.mode, c.rate, got, err)
		}
	}
}

func TestFailures(t *testing.T) {
	items := make([]string, 10000)
	for i := range items {
		items[i] = string(rune('a' + i%26))
	}
	if got := failures(items, 0, 1); len(got) != 0 {
		t.Fatalf("rate 0 failed %d items", len(got))
	}
	if got := failures(items, 1, 1); len(got) != len(items) {
		t.Fatalf("rate 1 failed %d of %d", len(got), len(items))
	}
	got := failures(items, 0.1, 1)
	if frac := float64(len(got)) / float64(len(items)); math.Abs(frac-0.1) > 0.02 {
		t.Fatalf("rate 0.1 failed %.3f of items", frac)
	}
	kinds := map[string]int{}
	for _, r := range got {
		kinds[r.Status]++
	}
	if kinds["retry"] == 0 || kinds["failed"] == 0 || len(kinds) != 2 {
		t.Fatalf("statuses = %v", kinds)
	}
}

package main

import (
	"flag"
	"reflect"
	"testing"
)

func TestParseInterspersed(t *testing.T) {
	cases := []struct {
		name      string
		argv      []string
		wantWrite bool
		wantBatch string
		wantPos   []string
	}{
		{
			name:      "flags before positional",
			argv:      []string{"--write", "--batch", "theme/", "extra"},
			wantWrite: true, wantBatch: "theme/", wantPos: []string{"extra"},
		},
		{
			name:      "flag after positional (the footgun)",
			argv:      []string{"page.json", "--write"},
			wantWrite: true, wantBatch: "", wantPos: []string{"page.json"},
		},
		{
			name:      "flags interleaved with positionals",
			argv:      []string{"a", "--write", "b", "--batch", "t/"},
			wantWrite: true, wantBatch: "t/", wantPos: []string{"a", "b"},
		},
		{
			name:    "no flags",
			argv:    []string{"just", "words"},
			wantPos: []string{"just", "words"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("review", flag.ContinueOnError)
			write := fs.Bool("write", false, "")
			batch := fs.String("batch", "", "")
			got := parseInterspersed(fs, tc.argv)
			if *write != tc.wantWrite {
				t.Errorf("write = %v, want %v", *write, tc.wantWrite)
			}
			if *batch != tc.wantBatch {
				t.Errorf("batch = %q, want %q", *batch, tc.wantBatch)
			}
			if !reflect.DeepEqual(got, tc.wantPos) {
				t.Errorf("positionals = %v, want %v", got, tc.wantPos)
			}
		})
	}
}

package parse

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestParseCuesBasic(t *testing.T) {
	in := []byte("[00:00:00.000 --> 00:00:02.500] Hello world.\n" +
		"[00:00:02.500 --> 00:00:05.000] This is a test.\n")
	cues, err := ParseCues(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 {
		t.Fatalf("got %d cues, want 2", len(cues))
	}
	if !approxEq(cues[0].Start, 0) || !approxEq(cues[0].End, 2.5) {
		t.Errorf("cue 0 times: %v -> %v", cues[0].Start, cues[0].End)
	}
	if cues[0].Text != "Hello world." {
		t.Errorf("cue 0 text: %q", cues[0].Text)
	}
	if !approxEq(cues[1].Start, 2.5) || !approxEq(cues[1].End, 5.0) {
		t.Errorf("cue 1 times: %v -> %v", cues[1].Start, cues[1].End)
	}
}

func TestParseCuesMillis(t *testing.T) {
	in := []byte("[01:23:45.678 --> 01:23:50.123] Foo.\n")
	cues, err := ParseCues(in)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := float64(1*3600+23*60+45) + 0.678
	wantEnd := float64(1*3600+23*60+50) + 0.123
	if !approxEq(cues[0].Start, wantStart) {
		t.Errorf("start: got %v want %v", cues[0].Start, wantStart)
	}
	if !approxEq(cues[0].End, wantEnd) {
		t.Errorf("end: got %v want %v", cues[0].End, wantEnd)
	}
}

func TestParseCuesEmptyInput(t *testing.T) {
	cues, err := ParseCues(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 0 {
		t.Errorf("expected 0 cues, got %d", len(cues))
	}
}

func TestParseCuesBlankLinesIgnored(t *testing.T) {
	in := []byte("\n[00:00:00.000 --> 00:00:01.000] One.\n\n[00:00:01.000 --> 00:00:02.000] Two.\n\n")
	cues, err := ParseCues(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 {
		t.Fatalf("got %d cues", len(cues))
	}
}

func TestParseCuesCRLF(t *testing.T) {
	in := []byte("[00:00:00.000 --> 00:00:01.000] One.\r\n[00:00:01.000 --> 00:00:02.000] Two.\r\n")
	cues, err := ParseCues(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 || cues[0].Text != "One." || cues[1].Text != "Two." {
		t.Fatalf("CRLF parse failure: %+v", cues)
	}
}

func TestParseCuesEmptyText(t *testing.T) {
	in := []byte("[00:00:00.000 --> 00:00:01.000] \n")
	cues, err := ParseCues(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 1 || cues[0].Text != "" {
		t.Fatalf("expected empty text cue, got %+v", cues)
	}
}

func TestParseCuesNoHours(t *testing.T) {
	in := []byte("[00:05.600 --> 00:10.000] Short form.\n[12:34.567 --> 12:35.000] Still no hour.\n")
	cues, err := ParseCues(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 {
		t.Fatalf("got %d cues", len(cues))
	}
	if !approxEq(cues[0].Start, 5.6) || !approxEq(cues[0].End, 10.0) {
		t.Errorf("cue 0 times: %v -> %v", cues[0].Start, cues[0].End)
	}
	if !approxEq(cues[1].Start, 12*60+34.567) {
		t.Errorf("cue 1 start: %v", cues[1].Start)
	}
}

func TestParseCuesNonCueLineIgnored(t *testing.T) {
	in := []byte("[00:00:00.000 --> 00:00:01.000] ok\nTranscription results written to '/some/path'\n")
	cues, err := ParseCues(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 1 {
		t.Fatalf("expected 1 cue, got %d", len(cues))
	}
}

func TestParseCuesMalformedBracketed(t *testing.T) {
	in := []byte("[00:00:00.000 --> 00:00:01.000] ok\n[bad cue line]\n")
	if _, err := ParseCues(in); err == nil {
		t.Fatal("expected error on malformed bracketed line")
	}
}

func TestParseCuesFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "sample.txt"))
	if err != nil {
		t.Fatal(err)
	}
	cues, err := ParseCues(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 4 {
		t.Fatalf("expected 4 cues, got %d", len(cues))
	}
	if cues[3].Text != "Way later in the episode." {
		t.Errorf("last cue text wrong: %q", cues[3].Text)
	}
	if !approxEq(cues[3].Start, float64(1*3600+23*60+45)+0.678) {
		t.Errorf("last cue start wrong: %v", cues[3].Start)
	}
}

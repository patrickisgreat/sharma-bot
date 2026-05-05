package parse

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"sharma-bot/internal/state"
)

type Cue struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type CuesFile struct {
	Cues []Cue `json:"cues"`
}

var cueRE = regexp.MustCompile(`^\[(?:(\d{2}):)?(\d{2}):(\d{2})\.(\d{3}) --> (?:(\d{2}):)?(\d{2}):(\d{2})\.(\d{3})\] ?(.*)$`)

func Run(corpusDir string) error {
	db, err := state.Open(filepath.Join(corpusDir, "state.db"))
	if err != nil {
		return err
	}
	defer db.Close()

	pending, err := state.Pending(db, state.StageDiscovered)
	if err != nil {
		return err
	}

	var ok, fail int
	for _, ep := range pending {
		rawPath := filepath.Join(corpusDir, ep.RawPath)
		outPath := filepath.Join(corpusDir, "cues", ep.Source, ep.ExternalID+".json")
		if err := parseOne(rawPath, outPath); err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", ep.ID, err)
			_ = state.SetStage(db, ep.ID, state.StageFailed, err.Error())
			fail++
			continue
		}
		if err := state.SetStage(db, ep.ID, state.StageParsed, ""); err != nil {
			return err
		}
		ok++
	}
	fmt.Printf("parse: %d ok, %d failed (of %d pending)\n", ok, fail, len(pending))
	return nil
}

func parseOne(in, out string) error {
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	cues, err := ParseCues(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(CuesFile{Cues: cues})
}

func ParseCues(data []byte) ([]Cue, error) {
	cues := []Cue{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			continue
		}
		if line[0] != '[' {
			continue
		}
		m := cueRE.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("line %d: not a cue: %q", lineNo, line)
		}
		cues = append(cues, Cue{
			Start: ts(m[1], m[2], m[3], m[4]),
			End:   ts(m[5], m[6], m[7], m[8]),
			Text:  m[9],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cues, nil
}

func ts(hh, mm, ss, ms string) float64 {
	h, _ := strconv.Atoi(hh)
	m, _ := strconv.Atoi(mm)
	s, _ := strconv.Atoi(ss)
	millis, _ := strconv.Atoi(ms)
	return float64(h*3600+m*60+s) + float64(millis)/1000.0
}

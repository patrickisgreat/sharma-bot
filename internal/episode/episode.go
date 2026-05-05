package episode

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var filenameRE = regexp.MustCompile(`^(\d{8})(?:_S(\d+)_Ep?(\d+))?_+(.+?)-ep_([a-z0-9]+)\.txt$`)

type Meta struct {
	Date       string
	Season     int
	EpisodeNum int
	TitleSlug  string
	ExternalID string
}

func ParseFilename(name string) (Meta, error) {
	m := filenameRE.FindStringSubmatch(name)
	if m == nil {
		return Meta{}, fmt.Errorf("filename does not match expected format: %s", name)
	}
	raw := m[1]
	meta := Meta{
		Date:       raw[0:4] + "-" + raw[4:6] + "-" + raw[6:8],
		TitleSlug:  strings.Trim(m[4], "_"),
		ExternalID: m[5],
	}
	if m[2] != "" {
		meta.Season, _ = strconv.Atoi(m[2])
		meta.EpisodeNum, _ = strconv.Atoi(m[3])
	}
	return meta, nil
}

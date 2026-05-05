package episode

import "testing"

func TestParseFilenameWithSeasonE(t *testing.T) {
	// Older-style "_E1" form.
	m, err := ParseFilename("20220720_S1_E1__Nik_Sharma_and_Moiz_Ali__Why_Native_Rejected_Investors-ep_9lmar28vdkl4r2nw.txt")
	if err != nil {
		t.Fatal(err)
	}
	if m.Date != "2022-07-20" {
		t.Errorf("date: %q", m.Date)
	}
	if m.Season != 1 || m.EpisodeNum != 1 {
		t.Errorf("season/ep: %d/%d", m.Season, m.EpisodeNum)
	}
	if m.ExternalID != "9lmar28vdkl4r2nw" {
		t.Errorf("external id: %q", m.ExternalID)
	}
	if m.TitleSlug != "Nik_Sharma_and_Moiz_Ali__Why_Native_Rejected_Investors" {
		t.Errorf("title slug: %q", m.TitleSlug)
	}
}

func TestParseFilenameWithSeasonEp(t *testing.T) {
	// Newer-style "_Ep12" form.
	m, err := ParseFilename("20240115_S3_Ep12__Some_Title-ep_abc123.txt")
	if err != nil {
		t.Fatal(err)
	}
	if m.Season != 3 || m.EpisodeNum != 12 {
		t.Errorf("season/ep: %d/%d", m.Season, m.EpisodeNum)
	}
	if m.TitleSlug != "Some_Title" {
		t.Errorf("title slug: %q", m.TitleSlug)
	}
}

func TestParseFilenameNoSeason(t *testing.T) {
	// Filenames without season/episode metadata still parse.
	m, err := ParseFilename("20230501__A_Title_Slug-ep_xyz789.txt")
	if err != nil {
		t.Fatal(err)
	}
	if m.Date != "2023-05-01" {
		t.Errorf("date: %q", m.Date)
	}
	if m.Season != 0 || m.EpisodeNum != 0 {
		t.Errorf("expected zero season/ep, got %d/%d", m.Season, m.EpisodeNum)
	}
	if m.ExternalID != "xyz789" {
		t.Errorf("external id: %q", m.ExternalID)
	}
	if m.TitleSlug != "A_Title_Slug" {
		t.Errorf("title slug: %q", m.TitleSlug)
	}
}

func TestParseFilenameMultiUnderscoreTitle(t *testing.T) {
	// Real corpus filenames have many underscores in the title and the
	// `__` separator between season/ep block and title is preserved as
	// part of the slug after Trim.
	m, err := ParseFilename("20220727_S1_E2__Why_Nik_Sharma_Bought_Long_Wknd___How_to_Optimize-ep_5lbvj5qadp4oj682.txt")
	if err != nil {
		t.Fatal(err)
	}
	if m.TitleSlug != "Why_Nik_Sharma_Bought_Long_Wknd___How_to_Optimize" {
		t.Errorf("title slug: %q", m.TitleSlug)
	}
	if m.ExternalID != "5lbvj5qadp4oj682" {
		t.Errorf("external id: %q", m.ExternalID)
	}
}

func TestParseFilenameRejectsBadFormat(t *testing.T) {
	cases := []string{
		"random.txt",
		"20220720_no_external_id.txt",
		"20220720__Title-ep_BADCASE.txt", // uppercase external id not allowed
		"2022_S1_E1__Title-ep_abc.txt",   // non-8-digit date
		"20220720_S1_E1__Title.txt",      // missing -ep_
	}
	for _, name := range cases {
		if _, err := ParseFilename(name); err == nil {
			t.Errorf("expected error for %q", name)
		}
	}
}

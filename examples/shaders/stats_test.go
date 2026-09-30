package main

import (
	"slices"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

func TestSummaryReportsMeanMedianAndTail(t *testing.T) {
	var s samples
	for _, ms := range []int{5, 1, 3, 2, 9} {
		s.add(time.Duration(ms) * time.Millisecond)
	}
	got := s.summary()
	want := stageSummary{Count: 5, MeanMS: 4, P50MS: 3, P95MS: 9}
	if got != want {
		t.Fatalf("summary = %+v, want %+v", got, want)
	}
}

func TestSummaryOfNoSamplesIsZero(t *testing.T) {
	var s samples
	if got := s.summary(); got != (stageSummary{}) {
		t.Fatalf("summary = %+v, want zero", got)
	}
}

func TestSamplesKeepOnlyTheRecentWindow(t *testing.T) {
	var s samples
	// Startup outliers must age out, or a long run never reflects steady state.
	for range 10 {
		s.add(time.Second)
	}
	for range sampleWindow {
		s.add(2 * time.Millisecond)
	}
	got := s.summary()
	want := stageSummary{Count: sampleWindow, MeanMS: 2, P50MS: 2, P95MS: 2}
	if got != want {
		t.Fatalf("summary = %+v, want %+v", got, want)
	}
}

func TestQueryArgsSelectOnlyKnownFlags(t *testing.T) {
	known := func(name string) bool { return name == "preset" || name == "density" || name == "mosaic" }
	cases := []struct {
		query string
		want  []string
	}{
		{"", nil},
		{"?", nil},
		{"?preset=julia&density=24", []string{"-density=24", "-preset=julia"}},
		{"?mosaic", []string{"-mosaic=true"}},
		{"?preset=flowing%20noise", []string{"-preset=flowing noise"}},
		// A page may carry unrelated parameters; they must not abort flag parsing.
		{"?utm_source=x&preset=orbits", []string{"-preset=orbits"}},
		{"?preset=%zz", nil},
	}
	for _, c := range cases {
		if got := queryArgs(c.query, known); !slices.Equal(got, c.want) {
			t.Errorf("queryArgs(%q) = %q, want %q", c.query, got, c.want)
		}
	}
}

func TestParseMedium(t *testing.T) {
	for name, want := range map[string]picture.KittyMedium{"shm": picture.KittyMediumSharedMemory, "direct": picture.KittyMediumDirect} {
		if got, err := parseMedium(name); err != nil || got != want {
			t.Errorf("parseMedium(%q) = %v, %v; want %v", name, got, err, want)
		}
	}
	if _, err := parseMedium("carrier-pigeon"); err == nil {
		t.Error("an unknown medium was accepted")
	}
}

func TestReportIsPublishedOnlyWhereThereIsAPublisher(t *testing.T) {
	m, r := testModel(t)
	r.stages = fixedStages()
	if _, cmd := m.Update(reportMsg{}); cmd != nil {
		t.Fatal("without a publisher the report timer should stop")
	}
	var published []report
	m.publish = func(r report) { published = append(published, r) }
	_, render := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(render())
	_, again := m.Update(reportMsg{})
	if len(published) != 1 || published[0].Stages["map"].P50MS != 3 {
		t.Fatalf("published = %+v, want one report carrying the frame's stages", published)
	}
	if again == nil {
		t.Fatal("the report timer did not reschedule")
	}
}

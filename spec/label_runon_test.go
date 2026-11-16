// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// lastLine is the bottom row of a view, where vertical bar labels sit.
func lastLine(view string) string {
	lines := strings.Split(strings.TrimRight(ansi.Strip(view), "\n"), "\n")
	return lines[len(lines)-1]
}

// barsLabelledEvery is a bar chart of n one-column bars whose labels are set
// on every k-th bar (1-based numbers) and empty between.
func barsLabelledEvery(n, k, width int) Spec {
	s := Spec{Type: ChartTypeBar, Width: width, Height: 8}
	var values []DataPoint
	for i := 1; i <= n; i++ {
		label := ""
		if i == 1 || i%k == 0 {
			label = strconv.Itoa(i)
		}
		s.XAxis.Labels = append(s.XAxis.Labels, label)
		values = append(values, DataPoint{Y: float64(i)})
	}
	s.Data.Series = []Series{{Name: "v", Values: values}}
	return s
}

func TestBarLabelRunsOnUnderUnlabelledBars(t *testing.T) {
	// 30 bars in 60 columns: each bar is one column wide.
	m, err := Build(barsLabelledEvery(30, 5, 60))
	if err != nil {
		t.Fatal(err)
	}
	row := lastLine(m.(interface{ View() string }).View())
	for _, want := range []string{"1", "5", "10", "15", "20", "25", "30"} {
		if !strings.Contains(" "+row+" ", " "+want+" ") {
			t.Fatalf("label %q not drawn whole in %q", want, row)
		}
	}
}

func TestBarLabelsAreStillBoundByTheirBarWhenAllAreSet(t *testing.T) {
	s := barsLabelledEvery(30, 1, 60)
	m, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	row := lastLine(m.(interface{ View() string }).View())
	// Every bar is labelled, so "10" is cut to its one-column bar, as before.
	if strings.Contains(row, "10") {
		t.Fatalf("a label ran into a labelled neighbour: %q", row)
	}
}

func TestBarLabelsCutByWidthNotBytes(t *testing.T) {
	s := Spec{Type: ChartTypeBar, Width: 12, Height: 6, XAxis: XAxis{Labels: []string{"1951–1980", "b"}},
		Data: Data{Series: []Series{{Name: "v", Values: []DataPoint{{Y: 1}, {Y: 2}}}}}}
	m, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if row := lastLine(m.(interface{ View() string }).View()); strings.ContainsRune(row, '\uFFFD') {
		t.Fatalf("a multibyte label was cut mid-rune: %q", row)
	}
}

package wavelinechart

import (
	"testing"
)

// When xStep == 0 the X axis is hidden, GraphHeight() == h, but
// Origin().Y == h-1. The Y scale factor must use GraphHeight() (matching
// the streamlinechart #7 fix) so pushed values don't lose a row of resolution.
func TestNewDataSet_YScaleUsesGraphHeight_WhenXStepZero(t *testing.T) {
	const h = 10
	m := New(20, h, WithXYSteps(0, 2))

	if m.GraphHeight() != h {
		t.Fatalf("precondition: expected GraphHeight=%d with xStep=0, got %d", h, m.GraphHeight())
	}
	if m.Origin().Y != h-1 {
		t.Fatalf("precondition: expected Origin().Y=%d, got %d", h-1, m.Origin().Y)
	}

	want := float64(m.GraphHeight()) / (m.ViewMaxY() - m.ViewMinY())
	got := m.dSets[DefaultDataSetName].pBuf.Scale().Y
	if got != want {
		t.Fatalf("newDataSet Y scale = %v, want %v (GraphHeight()/range, not Origin().Y/range)", got, want)
	}
}

// rescaleData (called by SetViewYRange/Resize/etc.) must apply the same
// GraphHeight()-based factor.
func TestRescaleData_YScaleUsesGraphHeight_WhenXStepZero(t *testing.T) {
	const h = 10
	m := New(20, h, WithXYSteps(0, 2), WithYRange(0, 10))

	// Trigger rescaleData by narrowing the view range.
	m.SetViewYRange(0, 5)

	want := float64(m.GraphHeight()) / (m.ViewMaxY() - m.ViewMinY())
	got := m.dSets[DefaultDataSetName].pBuf.Scale().Y
	if got != want {
		t.Fatalf("rescaled Y scale = %v, want %v", got, want)
	}
}

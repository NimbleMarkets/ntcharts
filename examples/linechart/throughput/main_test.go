package main

import (
	"testing"
	"time"
)

func TestRollingWindowDropsSpike(t *testing.T) {
	m := newModel()
	if m.chart.ViewMinY() != 0 || m.chart.ViewMaxY() != 3500 {
		t.Fatalf("initial range = %v..%v, want 0..3500", m.chart.ViewMinY(), m.chart.ViewMaxY())
	}
	now := time.UnixMilli(int64(m.chart.ViewMaxX() * 1e3))
	for i := 0; i < 11; i++ {
		now = now.Add(time.Second)
		m.push(now)
		m.refresh(now)
	}
	if m.chart.ViewMinY() != 0 || m.chart.ViewMaxY() > 100 {
		t.Fatalf("expired spike still affects range: %v..%v", m.chart.ViewMinY(), m.chart.ViewMaxY())
	}
	// The extra interpolation sample is retained, then expires next tick.
	if removed := m.chart.TrimBefore(now.Add(-time.Minute)); removed != 2 {
		t.Fatalf("expected one preceding sample per dataset, removed %d", removed)
	}
	// After discarding all samples, an empty fit leaves the last range intact.
	if removed := m.chart.TrimBefore(now.Add(time.Second)); removed != 122 {
		t.Fatalf("retention grew beyond 61 points per dataset: removed %d", removed)
	}
}

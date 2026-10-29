package spec

import (
	"encoding/json"
	"fmt"
	"math"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

// MarkMetadata preserves a producer's stable identity and original field values.
// IDs must be unique within a chart. Fields must contain JSON-compatible values.
// Aggregate marks describe their aggregate, not an arbitrary contributing row.
type MarkMetadata struct {
	ID     string         `json:"id"`
	Fields map[string]any `json:"fields,omitempty"`
}

// Mark identifies an inspectable observation. Series and Point are zero-based;
// heatmap marks use Series=-1 and Point indexes the flattened row-major cells.
type Mark struct {
	MarkMetadata
	Kind          ChartType
	Series, Point int
}

// InspectionMsg reports keyboard focus or mouse inspection without selecting.
type InspectionMsg struct{ Mark Mark }

// SelectionMsg is a snapshot of all selected marks, in chart order. Empty means
// cleared. Field maps are owned by the recipient, never shared with the model.
type SelectionMsg struct{ Marks []Mark }

// InteractiveModel adds keyboard inspection, mark selection and local-coordinate
// picking to a static chart. Build retains its existing return types; callers
// explicitly opt in through BuildInteractive. Call Update and retain its Cmd to
// deliver InspectionMsg/SelectionMsg to the host. This is a pointer model.
type InteractiveModel struct {
	spec     Spec
	marks    []Mark
	cells    map[canvas.Point][]int
	selected map[string]bool
	focus    int
	model    interface{ View() string }
}

// BuildInteractive supports ordinary bar, scatter and heatmap charts. Combos
// are rejected. Marks with metadata use producer IDs; without metadata IDs are
// positional and are stable only while data order is unchanged. If any mark
// has metadata, missing metadata denotes a synthetic/noninteractive datum.
func BuildInteractive(s Spec) (*InteractiveModel, error) {
	if s.Type != ChartTypeBar && s.Type != ChartTypeScatter && s.Type != ChartTypeHeatmap || IsCategoricalCombo(s) {
		return nil, fmt.Errorf("spec: interactive rendering supports bar, scatter and heatmap")
	}
	if _, err := Build(s); err != nil {
		return nil, err
	}
	// Take ownership of all nested data, including arbitrary field maps.
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var owned Spec
	if err := json.Unmarshal(raw, &owned); err != nil {
		return nil, err
	}
	if owned.Heat != nil && len(owned.Heat.Matrix) > 0 {
		owned.Heat.Cells = nil
		for y, row := range owned.Heat.Matrix {
			for x, z := range row {
				owned.Heat.Cells = append(owned.Heat.Cells, HeatCell{X: float64(x), Y: float64(y), Z: z})
			}
		}
		owned.Heat.Matrix = nil
	}
	if owned.Type == ChartTypeHeatmap {
		owned.Data.Series = nil
	} else {
		owned.Heat = nil
	}
	m := &InteractiveModel{spec: owned, focus: -1, selected: map[string]bool{}}
	hasMetadata := false
	for _, s := range owned.Data.Series {
		for _, p := range s.Values {
			hasMetadata = hasMetadata || p.Mark != nil
		}
	}
	if owned.Heat != nil {
		for _, p := range owned.Heat.Cells {
			hasMetadata = hasMetadata || p.Mark != nil
		}
	}
	seen := map[string]bool{}
	add := func(meta *MarkMetadata, series, point int, fields map[string]any) error {
		if meta == nil && hasMetadata {
			return nil
		}
		if meta == nil {
			meta = &MarkMetadata{ID: fmt.Sprintf("%s/%d/%d", owned.Type, series, point), Fields: fields}
		}
		if meta.ID == "" || seen[meta.ID] {
			return fmt.Errorf("spec: interactive marks require nonempty unique IDs: %q", meta.ID)
		}
		seen[meta.ID] = true
		m.marks = append(m.marks, Mark{MarkMetadata: *meta, Kind: owned.Type, Series: series, Point: point})
		return nil
	}
	for i, s := range owned.Data.Series {
		for j, p := range s.Values {
			if err := add(p.Mark, i, j, map[string]any{"x": PointX(p, j, owned.Data.XAxisData), "y": p.Y, "series": s.Name}); err != nil {
				return nil, err
			}
		}
	}
	if owned.Type == ChartTypeHeatmap {
		for j, p := range owned.Heat.Cells {
			if err := add(p.Mark, -1, j, map[string]any{"x": p.X, "y": p.Y, "z": p.Z}); err != nil {
				return nil, err
			}
		}
	}
	if err := m.draw(); err != nil {
		return nil, err
	}
	return m, nil
}

func cloneMark(mark Mark) Mark {
	b, _ := json.Marshal(mark.Fields)
	mark.Fields = nil
	_ = json.Unmarshal(b, &mark.Fields)
	return mark
}

// Marks returns independent metadata snapshots in keyboard traversal order.
func (m *InteractiveModel) Marks() []Mark {
	out := make([]Mark, len(m.marks))
	for i, v := range m.marks {
		out[i] = cloneMark(v)
	}
	return out
}

// Selection returns an independent selection snapshot.
func (m *InteractiveModel) Selection() SelectionMsg {
	out := SelectionMsg{Marks: []Mark{}}
	for _, v := range m.marks {
		if m.selected[v.ID] {
			out.Marks = append(out.Marks, cloneMark(v))
		}
	}
	return out
}

// Focused returns the inspected mark, if any.
func (m *InteractiveModel) Focused() (Mark, bool) {
	if m.focus < 0 {
		return Mark{}, false
	}
	return cloneMark(m.marks[m.focus]), true
}

// Highlight replaces selection atomically. Unknown IDs leave state unchanged.
func (m *InteractiveModel) Highlight(ids ...string) (SelectionMsg, error) {
	next := map[string]bool{}
	for _, id := range ids {
		found := false
		for _, v := range m.marks {
			if v.ID == id {
				found = true
				break
			}
		}
		if !found {
			return m.Selection(), fmt.Errorf("spec: unknown mark %q", id)
		}
		next[id] = true
	}
	m.selected = next
	_ = m.draw()
	return m.Selection(), nil
}

// Clear removes selection; keyboard inspection remains where it was.
func (m *InteractiveModel) Clear() SelectionMsg { s, _ := m.Highlight(); return s }

// Inspect focuses a mark without changing selection.
func (m *InteractiveModel) Inspect(id string) (InspectionMsg, error) {
	for i, v := range m.marks {
		if v.ID == id {
			m.focus = i
			_ = m.draw()
			return InspectionMsg{cloneMark(v)}, nil
		}
	}
	return InspectionMsg{}, fmt.Errorf("spec: unknown mark %q", id)
}

// HitTest returns every mark occupying the local canvas cell, in drawing order.
// Callers may cycle overlapping marks; keyboard traversal always reaches them.
func (m *InteractiveModel) HitTest(p canvas.Point) []Mark {
	out := []Mark{}
	for _, i := range m.cells[p] {
		out = append(out, cloneMark(m.marks[i]))
	}
	return out
}

// Resize rebuilds geometry while preserving focus and selection by ID. On an
// invalid/small viewport it leaves the previous chart and state unchanged.
func (m *InteractiveModel) Resize(w, h int) error {
	oldW, oldH := m.spec.Width, m.spec.Height
	m.spec.Width, m.spec.Height = w, h
	if err := m.draw(); err != nil {
		m.spec.Width, m.spec.Height = oldW, oldH
		return err
	}
	return nil
}
func (m *InteractiveModel) View() string { return m.model.View() }

// Update handles arrows/Tab (inspect), Home/End, Space/Enter (toggle), Escape
// (clear), and left mouse clicks. Mouse coordinates must be chart-local; hosts
// embedding charts translate their coordinates first. Key releases are ignored.
func (m *InteractiveModel) Update(msg tea.Msg) tea.Cmd {
	var event tea.Msg
	switch v := msg.(type) {
	case tea.KeyPressMsg:
		key := v.String()
		switch key {
		case "left", "up", "shift+tab", "right", "down", "tab", "home", "end":
			if len(m.marks) == 0 {
				return nil
			}
			n := m.focus
			switch key {
			case "home":
				n = 0
			case "end":
				n = len(m.marks) - 1
			case "left", "up", "shift+tab":
				if n < 0 {
					n = 0
				}
				n = (n - 1 + len(m.marks)) % len(m.marks)
			default:
				n = (n + 1) % len(m.marks)
			}
			e, _ := m.Inspect(m.marks[n].ID)
			event = e
		case "space", "enter":
			if m.focus < 0 {
				return nil
			}
			id := m.marks[m.focus].ID
			if m.selected[id] {
				delete(m.selected, id)
			} else {
				m.selected[id] = true
			}
			_ = m.draw()
			event = m.Selection()
		case "esc":
			event = m.Clear()
		}
	case tea.MouseClickMsg:
		mouse := v.Mouse()
		if mouse.Button != tea.MouseLeft {
			return nil
		}
		hits := m.cells[canvas.Point{X: mouse.X, Y: mouse.Y}]
		if len(hits) == 0 {
			return nil
		}
		// Repeated clicks cycle marks that occupy the same cell.
		n := hits[0]
		for j, i := range hits {
			if i == m.focus {
				n = hits[(j+1)%len(hits)]
				break
			}
		}
		m.focus = n
		id := m.marks[n].ID
		if m.selected[id] {
			delete(m.selected, id)
		} else {
			m.selected[id] = true
		}
		_ = m.draw()
		event = m.Selection()
	case tea.WindowSizeMsg:
		_ = m.Resize(v.Width, v.Height)
	}
	if event == nil {
		return nil
	}
	return func() tea.Msg { return event }
}

func (m *InteractiveModel) draw() error {
	built, err := Build(m.spec)
	if err != nil {
		return err
	}
	var c *canvas.Model
	switch b := built.(type) {
	case *barchart.Model:
		c = &b.Canvas
	case *linechart.Model:
		c = &b.Canvas
	case *CategoricalModel:
		c = &b.Canvas
	case *heatmap.Model:
		c = &b.Canvas
	default:
		return fmt.Errorf("spec: unsupported interactive model %T", built)
	}
	heatRows := len(m.spec.YAxis.Labels)
	if m.spec.Heat != nil {
		for _, p := range m.spec.Heat.Cells {
			heatRows = max(heatRows, int(math.Floor(p.Y))+1)
		}
	}
	cells := map[canvas.Point][]int{}
	add := func(p canvas.Point, i int) {
		if p.X >= 0 && p.Y >= 0 && p.X < c.Width() && p.Y < c.Height() {
			cells[p] = append(cells[p], i)
		}
	}
	if b, ok := built.(*barchart.Model); ok {
		indices := map[[2]int]int{}
		for i, v := range m.marks {
			segment := 0
			for j := 0; j < v.Series; j++ {
				if v.Point < len(m.spec.Data.Series[j].Values) {
					segment++
				}
			}
			indices[[2]int{v.Point, segment}] = i
		}
		for y := 0; y < c.Height(); y++ {
			for x := 0; x < c.Width(); x++ {
				p := canvas.Point{X: x, Y: y}
				bar, seg, ok := b.SegmentAt(p)
				if ok {
					if i, exists := indices[[2]int{bar, seg}]; exists {
						add(p, i)
					}
				}
			}
		}
	} else {
		for i, v := range m.marks {
			switch b := built.(type) {
			case *linechart.Model:
				ser := m.spec.Data.Series[v.Series]
				p := ser.Values[v.Point]
				y := p.Y
				if ser.YAxis == YAxisRight {
					y = b.Y2ToY(y)
				}
				if cp, ok := b.PointCell(canvas.Float64Point{X: resolveXFloat(m.spec, p, v.Point), Y: y}); ok {
					add(cp, i)
				}
			case *CategoricalModel:
				ser := b.data.Data.Series[v.Series]
				p := ser.Values[v.Point]
				centers := categoryCenters(b.GraphWidth(), len(b.labels))
				y := p.Y
				if ser.YAxis == YAxisRight {
					y = b.Y2ToY(y)
				}
				x := float64(len(b.labels)) * float64(centers[int(p.X.(float64))]) / float64(max(1, b.GraphWidth()-1))
				if cp, ok := b.PointCell(canvas.Float64Point{X: x, Y: y}); ok {
					add(cp, i)
				}
			case *heatmap.Model:
				p := m.spec.Heat.Cells[v.Point]
				ny := heatRows
				for _, cp := range b.CellPoints(heatmap.NewHeatPoint(p.X, float64(ny-1)-p.Y, p.Z)) {
					add(cp, i)
				}
			}
		}
	}
	for p, indices := range cells {
		selected, focused := false, false
		for _, i := range indices {
			selected = selected || m.selected[m.marks[i].ID]
			focused = focused || i == m.focus
		}
		cell := c.Cell(p)
		style := cell.Style
		if len(m.selected) > 0 && !selected {
			style = style.Faint(true)
		}
		if selected || focused {
			accent := lipgloss.Color("14") // cyan focus; yellow selection
			if selected {
				accent = lipgloss.Color("11")
			}
			style = style.Faint(false).Bold(true)
			if m.spec.Type == ChartTypeHeatmap {
				style = style.Background(accent)
			} else {
				style = style.Foreground(accent)
			}
		}
		c.SetCellStyle(p, style)
	}
	// Put an ASCII badge in one painted cell per active mark. Unlike underline
	// or reverse video on block glyphs, it remains legible without ANSI color.
	// Choose topmost/leftmost cells deterministically, independent of map order.
	anchors := map[int]canvas.Point{}
	for p, indices := range cells {
		for _, i := range indices {
			if !m.selected[m.marks[i].ID] && i != m.focus {
				continue
			}
			old, exists := anchors[i]
			if !exists || p.Y < old.Y || (p.Y == old.Y && p.X < old.X) {
				anchors[i] = p
			}
		}
	}
	badges := map[canvas.Point][2]bool{}
	for i, p := range anchors {
		flags := badges[p]
		flags[0] = flags[0] || m.selected[m.marks[i].ID]
		flags[1] = flags[1] || i == m.focus
		badges[p] = flags
	}
	for p, flags := range badges {
		glyph, accent := '>', lipgloss.Color("14")
		if flags[0] {
			glyph, accent = '*', lipgloss.Color("11")
		}
		if flags[0] && flags[1] {
			glyph = '@'
		}
		c.SetCell(p, canvas.NewCellWithStyle(glyph, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(accent)))
	}
	m.cells = cells
	m.model = built.(interface{ View() string })
	return nil
}

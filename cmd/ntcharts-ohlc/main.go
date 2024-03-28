// ntcharts - Copyright (c) 2024 Neomantra Corp.

// Displays OHLC data as a line chart from an input CSV file.
// The command can display the braille lines or continuous line and
// choose which lines to display.
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/canvas/runes"
	tslc "github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	spark "github.com/NimbleMarkets/ntcharts/sparkline"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var defaultStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")) // purple

var openLineStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("202")) // orange

var highLineStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("10")) // green

var lowLineStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("1")) // pink

var closeLineStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")) // blue

var axisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3")) // yellow

var labelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6")) // cyan

const ( // used for flag options and data set names
	OpenOptionName     = "open"
	HighOptionName     = "high"
	LowOptionName      = "low"
	CloseOptionName    = "close"
	AdjCloseOptionName = "adjclose"
	VolumeOptionName   = "vol"
)

const mil = 1000000.0
const daySeconds = 86400

var dataSetStyles = map[string]lipgloss.Style{
	OpenOptionName:     openLineStyle,
	HighOptionName:     highLineStyle,
	LowOptionName:      lowLineStyle,
	CloseOptionName:    closeLineStyle,
	AdjCloseOptionName: closeLineStyle,
}

// displayOptions contains which OHLC lines to display
type displayOptions struct {
	All        bool
	Open       bool
	High       bool
	Low        bool
	Close      bool
	Volume     bool
	AdjClose   bool // replaces Close with Adjusted Close
	LineStyle  runes.LineStyle
	UseBraille bool // whether to draw braille lines
}

var displayOpts displayOptions

// record represents a record entry in the CSV file
type record struct {
	Date          time.Time
	Open          float64
	High          float64
	Low           float64
	Close         float64
	AdjustedClose float64
	Volume        int64
}

// NewRecord returns a record from a given []string
// containing OHLC record information
// Function expects the following columns:
// [Date, Open, High, Low, Close, Adj Close, Volume]
func NewRecord(s []string) (r record) {
	r = record{}
	if len(s) < 7 || strings.ToLower(s[0]) == `date` {
		return
	}
	var err error
	r.Date, err = time.Parse("2006-01-02", s[0])
	if err != nil {
		log.Fatalf("Wrong CSV Date format for value: %s\n", s[0])
		return
	}
	r.Open, err = strconv.ParseFloat(s[1], 64)
	if err != nil {
		log.Fatalf("Wrong CSV Open value: %s\n", s[1])
		return
	}
	r.Open, err = strconv.ParseFloat(s[1], 64)
	if err != nil {
		log.Fatalf("Wrong CSV Open value: %s\n", s[1])
		return
	}
	r.High, err = strconv.ParseFloat(s[2], 64)
	if err != nil {
		log.Fatalf("Wrong CSV High value: %s\n", s[2])
		return
	}
	r.Low, err = strconv.ParseFloat(s[3], 64)
	if err != nil {
		log.Fatalf("Wrong CSV Low value: %s\n", s[3])
		return
	}
	r.Close, err = strconv.ParseFloat(s[4], 64)
	if err != nil {
		log.Fatalf("Wrong CSV Close value: %s\n", s[4])
		return
	}
	r.AdjustedClose, err = strconv.ParseFloat(s[5], 64)
	if err != nil {
		log.Fatalf("Wrong CSV AdjustedClose value: %s\n", s[5])
		return
	}
	r.Volume, err = strconv.ParseInt(s[6], 10, 64)
	if err != nil {
		log.Fatalf("Wrong CSV Volume value: %s\n", s[6])
		return
	}
	return
}

type model struct {
	chart       tslc.Model
	sparkline   spark.Model
	zoneManager *zone.Manager

	vol  map[int64]float64
	minV float64
	maxV float64
}

func newModel(minTime, maxTime time.Time, minY, maxY float64, tsm map[string][]tslc.TimePoint) *model {
	m := model{
		chart: tslc.New(20, 10,
			tslc.WithTimeRange(minTime, maxTime),
			tslc.WithYRange(minY, maxY),
			tslc.WithAxesStyles(axisStyle, labelStyle),
		),
		sparkline:   spark.New(20, 10),
		zoneManager: zone.New(),
		vol:         make(map[int64]float64),
		minV:        float64(minTime.Unix()),
		maxV:        float64(maxTime.Unix()),
	}
	m.chart.Focus()

	// set time series data for each line
	for name, tsd := range tsm {
		if name == VolumeOptionName {
			for _, p := range tsd {
				m.vol[p.Time.Unix()] = p.Value
				switch {
				case p.Value < m.minV:
					m.minV = p.Value
				case p.Value > m.maxV:
					m.maxV = p.Value
				}
			}
		} else {
			m.chart.SetDataSetStyle(name, dataSetStyles[name])
			if !displayOpts.UseBraille {
				m.chart.SetDataSetLineStyle(name, displayOpts.LineStyle)
			}
			for _, p := range tsd {
				m.chart.PushDataSet(name, p)
			}
		}
	}

	// set X values such that each column is a single day
	m.resetTimeRange()

	// replace default update handler with handler that
	// moves graph left and right with mouse wheel
	// incrementing by 10 days at a time
	m.chart.UpdateHandler = tslc.DateNoZoomUpdateHandler(10)

	// use bubblezone to handle mouse events
	m.chart.SetZoneManager(m.zoneManager)
	return &m
}

// resetTimeRange set displayed time range such that each graph column is a single day
func (m *model) resetTimeRange() {
	viewMin := time.Unix(int64(m.chart.ViewMinX()), 0)
	viewMax := viewMin.Add(time.Hour * time.Duration(24*m.chart.GraphWidth()))
	if viewMax.Unix() > int64(m.chart.MaxX()) {
		viewMax = time.Unix(int64(m.chart.MaxX()), 0)
	}
	m.chart.SetViewTimeRange(viewMin, viewMax)
}

// drawVolume draws the data set Volume onto the sparkline such that
// each column below the corresponding date for that volume
func (m *model) drawVolume() {
	m.sparkline.Clear()
	startX := int64(m.chart.ViewMinX())
	endX := int64(m.chart.ViewMaxX())
	for i := startX; i < endX; i += daySeconds {
		v := m.vol[i]
		if v != 0 {
			v -= m.minV
		}
		m.sparkline.Push(float64(v))
	}
	m.sparkline.Draw()
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// resize window to terminal screen sizes
		if displayOpts.Volume {
			cHeight := (msg.Height * 4 / 6)
			sHeight := (msg.Height * 1 / 6)
			extra := msg.Height - (sHeight + cHeight) - 5
			m.chart.Resize(msg.Width-2, cHeight+extra)
			m.resetTimeRange()
			m.sparkline.Resize(msg.Width-2, sHeight)
		} else {
			m.chart.Resize(msg.Width-2, msg.Height-4)
			m.resetTimeRange()
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.chart, _ = m.chart.Update(msg)
	if displayOpts.UseBraille {
		m.chart.DrawBrailleAll()
	} else {
		m.chart.DrawAll()
	}
	if displayOpts.Volume {
		m.drawVolume()
	}
	return m, nil
}

func (m model) View() string {
	var legend string
	if displayOpts.All || displayOpts.Open {
		legend += openLineStyle.Render(" OPEN")
	}
	if displayOpts.All || displayOpts.High {
		legend += highLineStyle.Render(" HIGH")
	}
	if displayOpts.All || displayOpts.Low {
		legend += lowLineStyle.Render(" LOW")
	}
	if displayOpts.All || displayOpts.Close {
		if displayOpts.AdjClose {
			legend += closeLineStyle.Render(" ADJCLOSE")
		} else {
			legend += closeLineStyle.Render(" CLOSE")
		}
	}

	// combine line chart and sparkline if showing volume
	var graphView string
	if displayOpts.Volume {
		graphView = lipgloss.JoinVertical(lipgloss.Left,
			m.chart.View(),
			fmt.Sprintf("Daily Volume Range: %.02fM - %0.2fM\n", m.minV/mil, m.maxV/mil),
			m.sparkline.View(),
		)
	} else {
		graphView = m.chart.View()
	}

	startDate := time.Unix(int64(m.chart.MinX()), 0).UTC()
	endDate := time.Unix(int64(m.chart.MaxX()), 0).UTC()
	header := fmt.Sprintf("OHLC Chart from %s to %s. Legend [%s ]\n", startDate, endDate, legend)
	s := defaultStyle.Render(header + graphView)

	// wrap output string in bubblezone.Manager.Scan()
	// if SetZoneManager(bubblezone.Manager) is used
	return m.zoneManager.Scan(s)
}

// recordsFromCSV reads from a io.Reader and returns
// a slice of record objects
func recordsFromCSV(r io.Reader) (s []record) {
	csvReader := csv.NewReader(r)
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		newRecord := NewRecord(rec)
		if newRecord.Open != 0 {
			s = append(s, newRecord)
		}
	}
	return
}

func addOpen(r record, s map[string][]tslc.TimePoint, minY, maxY *float64) {
	if r.Open < *minY {
		*minY = r.Open
	}
	s[OpenOptionName] = append(s[OpenOptionName], tslc.TimePoint{Time: r.Date, Value: r.Open})
}

func addHigh(r record, s map[string][]tslc.TimePoint, minY, maxY *float64) {
	if r.High > *maxY {
		*maxY = r.High
	}
	s[HighOptionName] = append(s[HighOptionName], tslc.TimePoint{Time: r.Date, Value: r.High})
}

func addLow(r record, s map[string][]tslc.TimePoint, minY, maxY *float64) {
	if r.Low < *minY {
		*minY = r.Low
	}
	s[LowOptionName] = append(s[LowOptionName], tslc.TimePoint{Time: r.Date, Value: r.Low})
}

func addClose(r record, s map[string][]tslc.TimePoint, minY, maxY *float64) {
	if displayOpts.AdjClose {
		if r.AdjustedClose > *maxY {
			*maxY = r.AdjustedClose
		}
		s[AdjCloseOptionName] = append(s[AdjCloseOptionName], tslc.TimePoint{Time: r.Date, Value: r.Close})
	} else {
		if r.Close > *maxY {
			*maxY = r.Close
		}
		s[CloseOptionName] = append(s[CloseOptionName], tslc.TimePoint{Time: r.Date, Value: r.Close})
	}
}

func addVolume(r record, s map[string][]tslc.TimePoint) {
	s[VolumeOptionName] = append(s[VolumeOptionName], tslc.TimePoint{Time: r.Date, Value: float64(r.Volume)})
}

func timeseriesFromRecords(r []record) (s map[string][]tslc.TimePoint, minY float64, maxY float64, minTime, maxTime time.Time) {
	s = make(map[string][]tslc.TimePoint)
	if len(r) == 0 {
		return
	}
	minTime = r[0].Date
	maxTime = r[len(r)-1].Date
	// initialize min/max values
	switch {
	case displayOpts.All || displayOpts.Open:
		minY = r[0].Open
		maxY = r[0].Open
	case displayOpts.High:
		minY = r[0].High
		maxY = r[0].High
	case displayOpts.Low:
		minY = r[0].Low
		maxY = r[0].Low
	case displayOpts.Close:
		if displayOpts.AdjClose {
			minY = r[0].AdjustedClose
			maxY = r[0].AdjustedClose
		} else {
			minY = r[0].Close
			maxY = r[0].Close
		}
	}
	for _, rec := range r {
		if displayOpts.All || displayOpts.Open {
			addOpen(rec, s, &minY, &maxY)
		}
		if displayOpts.All || displayOpts.High {
			addHigh(rec, s, &minY, &maxY)
		}
		if displayOpts.All || displayOpts.Low {
			addLow(rec, s, &minY, &maxY)
		}
		if displayOpts.All || displayOpts.Close {
			addClose(rec, s, &minY, &maxY)
		}
		if displayOpts.Volume {
			addVolume(rec, s)
		}
	}
	return
}

func main() {
	var useThinStyle bool
	var filePath string
	flag.StringVar(&filePath, "filepath", "", "filepath to OHLC csv file, '-' to read from stdin")
	flag.BoolVar(&useThinStyle, "thin", false, "use thin lines (default: arc lines)")
	flag.BoolVar(&displayOpts.UseBraille, "braille", false, "use braille lines (default: arc lines)")
	flag.BoolVar(&displayOpts.Open, OpenOptionName, false, "whether to display OPEN line")
	flag.BoolVar(&displayOpts.High, HighOptionName, false, "whether to display HIGH line")
	flag.BoolVar(&displayOpts.Low, LowOptionName, false, "whether to display LOW line")
	flag.BoolVar(&displayOpts.Close, CloseOptionName, false, "whether to display CLOSE line")
	flag.BoolVar(&displayOpts.AdjClose, AdjCloseOptionName, false, "whether to replace CLOSE line with Adjusted CLOSE line (only used if --close enabled)")
	flag.BoolVar(&displayOpts.Volume, VolumeOptionName, false, "whether to display sparkline containing VOLUME")
	flag.Parse()

	// if nothing specified, default to display all lines
	if !displayOpts.Open && !displayOpts.High && !displayOpts.Low && !displayOpts.Close {
		displayOpts.All = true
	}
	if useThinStyle {
		displayOpts.LineStyle = runes.ThinLineStyle
	} else {
		displayOpts.LineStyle = runes.ArcLineStyle
	}

	// convert CSV rows into timeseries data
	var records []record
	switch filePath {
	case "":
		fmt.Println("Missing filepath")
		os.Exit(1)
	case "-":
		r := bufio.NewReader(os.Stdin)
		records = recordsFromCSV(r)
	default:
		f, err := os.Open(filePath)
		if err != nil {
			log.Fatal("Unable to read input file "+filePath, err)
		}
		defer f.Close()
		records = recordsFromCSV(f)
	}
	if len(records) == 0 {
		os.Exit(0)
	}
	ts, minY, maxY, minTime, maxTime := timeseriesFromRecords(records)

	// create model and start bubbletea Program
	m := newModel(minTime, maxTime, minY, maxY, ts)
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

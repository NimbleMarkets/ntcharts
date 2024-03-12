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

	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	tslc "github.com/NimbleMarkets/bubbletea-charts/linechart/timeserieslinechart"

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

var dataSetStyles = map[string]lipgloss.Style{
	"open":     openLineStyle,
	"high":     highLineStyle,
	"low":      lowLineStyle,
	"close":    closeLineStyle,
	"adjclose": closeLineStyle,
}

// displayOptions contains which OHLC lines to display
type displayOptions struct {
	All      bool
	Open     bool
	High     bool
	Low      bool
	Close    bool
	AdjClose bool
}

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
	opts        displayOptions
	zoneManager *zone.Manager
}

func NewModel(minTime, maxTime time.Time, minY, maxY float64, tsm map[string][]tslc.TimePoint, opts displayOptions) *model {
	min := minTime
	max := maxTime
	m := model{
		chart: tslc.New(20, 10,
			tslc.WithTimeRange(min, max),
			tslc.WithYRange(minY, maxY),
			tslc.WithAxesStyles(axisStyle, labelStyle),
		),
		zoneManager: zone.New(),
		opts:        opts,
	}
	m.chart.Focus()

	// set time series data for each line
	for name, tsd := range tsm {
		m.chart.SetDataSetStyles(name, runes.ArcLineStyle, dataSetStyles[name])
		for _, p := range tsd {
			m.chart.PushDataSet(name, p)
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

func (m model) Init() tea.Cmd {
	m.chart.DrawAll()
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// resize window to terminal screen sizes
		m.chart.Resize(msg.Width-3, msg.Height-4)
		m.resetTimeRange()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.chart, _ = m.chart.Update(msg)
	m.chart.DrawAll()
	return m, nil
}

func (m model) View() string {
	var legend string
	if m.opts.All || m.opts.Open {
		legend += openLineStyle.Render(" OPEN")
	}
	if m.opts.All || m.opts.High {
		legend += highLineStyle.Render(" HIGH")
	}
	if m.opts.All || m.opts.Low {
		legend += lowLineStyle.Render(" LOW")
	}
	if m.opts.All || m.opts.Close {
		if m.opts.AdjClose {
			legend += closeLineStyle.Render(" ADJCLOSE")
		} else {
			legend += closeLineStyle.Render(" CLOSE")
		}
	}

	startDate := time.Unix(int64(m.chart.MinX()), 0).UTC()
	endDate := time.Unix(int64(m.chart.MaxX()), 0).UTC()
	header := fmt.Sprintf("OHLC Chart from %s to %s. Legend [%s ]\n", startDate, endDate, legend)
	s := defaultStyle.Render(header + m.chart.View())

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
	s["open"] = append(s["open"], tslc.TimePoint{Time: r.Date, Value: r.Open})
}

func addHigh(r record, s map[string][]tslc.TimePoint, minY, maxY *float64) {
	if r.High > *maxY {
		*maxY = r.High
	}
	s["high"] = append(s["high"], tslc.TimePoint{Time: r.Date, Value: r.High})
}

func addLow(r record, s map[string][]tslc.TimePoint, minY, maxY *float64) {
	if r.Low < *minY {
		*minY = r.Low
	}
	s["low"] = append(s["low"], tslc.TimePoint{Time: r.Date, Value: r.Low})
}

func addClose(r record, s map[string][]tslc.TimePoint, minY, maxY *float64, useAdjClose bool) {
	if useAdjClose {
		if r.AdjustedClose > *maxY {
			*maxY = r.AdjustedClose
		}
		s["adjclose"] = append(s["adjclose"], tslc.TimePoint{Time: r.Date, Value: r.Close})
	} else {
		if r.Close > *maxY {
			*maxY = r.Close
		}
		s["close"] = append(s["close"], tslc.TimePoint{Time: r.Date, Value: r.Close})
	}
}

func timeseriesFromRecords(opts displayOptions, r []record) (s map[string][]tslc.TimePoint, minY float64, maxY float64, minTime, maxTime time.Time) {
	s = make(map[string][]tslc.TimePoint)
	if len(r) == 0 {
		return
	}
	minTime = r[0].Date
	maxTime = r[len(r)-1].Date
	// initialize min/max values
	switch {
	case opts.All || opts.Open:
		minY = r[0].Open
		maxY = r[0].Open
	case opts.High:
		minY = r[0].High
		maxY = r[0].High
	case opts.Low:
		minY = r[0].Low
		maxY = r[0].Low
	case opts.Close:
		if opts.AdjClose {
			minY = r[0].AdjustedClose
			maxY = r[0].AdjustedClose
		} else {
			minY = r[0].Close
			maxY = r[0].Close
		}
	}
	for _, rec := range r {
		if opts.All || opts.Open {
			addOpen(rec, s, &minY, &maxY)
		}
		if opts.All || opts.High {
			addHigh(rec, s, &minY, &maxY)
		}
		if opts.All || opts.Low {
			addLow(rec, s, &minY, &maxY)
		}
		if opts.All || opts.Close {
			addClose(rec, s, &minY, &maxY, opts.AdjClose)
		}
	}
	return
}

func main() {
	var displayOpts displayOptions
	var filePath string
	flag.StringVar(&filePath, "filepath", "", "filepath to OHLC csv file, '-' to read from stdin")
	flag.BoolVar(&displayOpts.Open, "open", false, "whether to display OPEN line")
	flag.BoolVar(&displayOpts.High, "high", false, "whether to display HIGH line")
	flag.BoolVar(&displayOpts.Low, "low", false, "whether to display LOW line")
	flag.BoolVar(&displayOpts.Close, "close", false, "whether to display CLOSE line")
	flag.BoolVar(&displayOpts.AdjClose, "adjclose", false, "whether to replace CLOSE line with Adjusted CLOSE line (only used if --close enabled)")
	flag.Parse()
	//TODO: if displaying volume, then it is a special column graph?

	// if nothing specified, default to display all lines
	if !displayOpts.Open && !displayOpts.High && !displayOpts.Low && !displayOpts.Close {
		displayOpts.All = true
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
	ts, minY, maxY, minTime, maxTime := timeseriesFromRecords(displayOpts, records)

	// create model and start bubbletea Program
	m := NewModel(minTime, maxTime, minY, maxY, ts, displayOpts)
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

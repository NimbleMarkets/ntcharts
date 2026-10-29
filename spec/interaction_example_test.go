package spec_test

import (
	"fmt"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

func ExampleBuildInteractive() {
	s := spec.Spec{Type: spec.ChartTypeBar, Width: 40, Height: 12,
		XAxis: spec.XAxis{Labels: []string{"Japan", "USA"}},
		Data: spec.Data{Series: []spec.Series{{Name: "Sales", Values: []spec.DataPoint{
			{Y: 42, Mark: &spec.MarkMetadata{ID: "jp", Fields: map[string]any{"Country": "Japan", "Sales": 42}}},
			{Y: 35, Mark: &spec.MarkMetadata{ID: "us", Fields: map[string]any{"Country": "USA", "Sales": 35}}},
		}}}},
	}
	chart, err := spec.BuildInteractive(s)
	if err != nil {
		panic(err)
	}
	inspection, _ := chart.Inspect("jp")
	fmt.Println(inspection.Mark.Fields["Country"])
	selection, _ := chart.Highlight("jp")
	fmt.Println(len(selection.Marks), selection.Marks[0].Fields["Sales"])
	fmt.Println(len(chart.Clear().Marks))
	// Output:
	// Japan
	// 1 42
	// 0
}

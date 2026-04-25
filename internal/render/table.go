package render

import (
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
)

// Table renders rows of arbitrary type into a table writer with the
// given header strings. Caller maps each item to a Row.
func Table(w io.Writer, header []string, rows [][]any) {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetStyle(table.StyleRounded)
	hdr := make(table.Row, len(header))
	for i, h := range header {
		hdr[i] = h
	}
	t.AppendHeader(hdr)
	for _, r := range rows {
		row := make(table.Row, len(r))
		for i, v := range r {
			row[i] = v
		}
		t.AppendRow(row)
	}
	t.Render()
}

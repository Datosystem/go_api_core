package table

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

type Row struct {
	table  *Table
	parent *Row
	child  *Row
	page   int
	startY float64
	height float64
	cells  []Cell
	hr     bool
}

func (r *Row) highestY() (hy float64) {
	for i, cell := range r.cells {
		colY := cell.endY
		style := cell.Style(&r.table.columns[i])
		if style != nil {
			colY += style.PaddingBottom
		}
		if hy < colY {
			hy = colY
		}
	}
	return
}

// Add the specified element and style on the row
func (r *Row) Add(value any, styles ...*Style) {
	style := &Style{}
	if len(styles) > 0 && styles[0] != nil {
		style = styles[0]
		r.cells[r.table.columnIndex].style = style
	} else if s := r.table.columns[r.table.columnIndex].Style; s != nil {
		style = s
	}

	colX := r.table.columnX()
	if r.cells[r.table.columnIndex].endX != 0 {
		colX = r.cells[r.table.columnIndex].endX
	}
	if fn, ok := value.(func()); ok {
		fn()
		_, fontHeight := r.table.pdf.GetFontSize()
		r.table.pdf.SetXY(colX, r.table.pdf.GetY()+fontHeight)
	} else {
		fRe, fGr, fBl := r.table.pdf.GetDrawColor()
		if style.Fill != nil && len(*style.Fill) == 3 {
			fill := *style.Fill
			r.table.pdf.SetFillColor(fill[0], fill[1], fill[2])
			r.table.pdf.SetDrawColor(fill[0], fill[1], fill[2])
		}
		tRe, tGr, tBl := r.table.pdf.GetTextColor()
		if style.Color != nil && len(*style.Color) == 3 {
			color := *style.Color
			r.table.pdf.SetTextColor(color[0], color[1], color[2])
		}

		var str string
		if reflect.TypeOf(value).Kind() == reflect.Ptr {
			rv := reflect.ValueOf(value)
			if rv.IsNil() {
				value = ""
			} else {
				value = rv.Elem().Interface()
			}
		}
		str = fmt.Sprintf("%v", value)
		if len(style.Format) > 0 {
			if style.Format == "-" {
				r.table.pdf.SetFontStyle("")
			} else {
				r.table.pdf.SetFontStyle(style.Format)
			}
		}
		if style.Size > 0 {
			r.table.pdf.SetFontSize(style.Size)
		}

		_, fontHeight := r.table.pdf.GetFontSize()

		colWd := r.table.columns[r.table.columnIndex].size
		if style.inline {
			colWd -= colX - r.table.columnX()
		} else {
			colWd -= style.PaddingLeft + style.PaddingRight
		}
		lines := r.table.pdf.SplitText(str, colWd)

		var margin float64

		firstLine := r.cells[r.table.columnIndex].endY == 0

		if len(style.VAlign) > 0 {
			rh := r.height
			hy := r.highestY()
			if rh != 0 || hy > r.startY {
				if rh == 0 {
					rh = hy - r.startY
				}
				rh = rh - style.PaddingTop - style.PaddingBottom
				margin = (rh - float64(len(lines))*(fontHeight+style.Ln)) / 2
				if style.VAlign == "M" {
					r.table.pdf.SetY(r.startY + margin)
				} else if style.VAlign == "S" {
					style.PaddingTop += margin
					style.PaddingBottom += margin
					margin = 0
				}
			}
		}

		for _, line := range lines {
			strWd := r.table.pdf.GetStringWidth(line)
			if style.Fill != nil && len(*style.Fill) == 3 {
				if !style.inline {
					r.table.pdf.Rect(r.table.columnX(), r.table.pdf.GetY(), r.table.originalColumnSize(r.table.columnIndex), fontHeight+style.Ln+style.PaddingTop+style.PaddingBottom, "FD")
				}
			}
			if firstLine {
				firstLine = false
				r.table.pdf.SetY(r.table.pdf.GetY() + style.PaddingTop)
			}
			if style.inline {
				r.table.pdf.SetX(colX + style.PaddingLeft)
				style.Align = "LT"
			} else {
				r.table.pdf.SetX(colX)
			}
			if style.inline && len(lines) == 1 {
				r.table.pdf.CellFormat(strWd+(r.table.pdf.GetCellMargin()*2), fontHeight+style.Ln, line, "", 0, style.Align, style.Fill != nil, 0, "")
				r.cells[r.table.columnIndex].endX = r.table.pdf.GetX() + style.PaddingRight
			} else {
				r.table.pdf.CellFormat(r.table.ColumnSize(), fontHeight+style.Ln, line, "", 2, style.Align, false, 0, "")
			}
			if style.inline {
				break
			}
		}

		if margin > 0 {
			r.table.pdf.SetXY(colX, r.table.pdf.GetY()+margin)
		}

		if style.Fill != nil {
			r.table.pdf.SetDrawColor(fRe, fGr, fBl)
		}

		if style.Color != nil {
			r.table.pdf.SetTextColor(tRe, tGr, tBl)
		}

		if len(style.Format) > 0 && r.table.columnIndex > 0 {
			s := r.cells[r.table.columnIndex-1].Style(&r.table.columns[r.table.columnIndex-1])
			if len(s.Format) == 0 || s.Format == "-" {
				r.table.pdf.SetFontStyle("")
			} else {
				r.table.pdf.SetFontStyle(s.Format)
			}
		}

		if style.inline && len(lines) > 1 {
			r.cells[r.table.columnIndex].endX = 0
			if style.Ln > 0 {
				r.table.pdf.SetY(r.table.pdf.GetY() + style.Ln)
			}
			r.table.Add(strings.Join(lines[1:], ""), style)
		}
	}
}

// Draws the row of the table
func (r *Row) Draw() {
	x := r.table.startX
	hy := math.Max(r.highestY(), r.startY+r.height)
	if hy < r.startY {
		return
	}

	if r.hr {
		r.table.pdf.Line(r.table.startX, hy, r.table.endX, hy)
		r.hr = false
	}

	for i := range r.table.columns {
		r.table.pdf.SetXY(x, r.startY)
		size := r.table.columns[i].size
		style := r.cells[i].Style(&r.table.columns[i])
		if style != nil {
			if style.Fill != nil {
				fill := *style.Fill
				r.table.pdf.SetFillColor(fill[0], fill[1], fill[2])
				if r.cells[i].endY < hy {
					startY := math.Max(r.startY, r.cells[i].endY)
					r.table.pdf.Rect(x, startY, size, hy-startY, "F")
				}
			}
			if len(style.Border) > 0 {
				r.table.DrawBorder(style.Border, x, r.startY, size, hy-r.startY)
			}
		}
		x += size
	}
	r.table.pdf.SetY(hy)
	if r.child != nil {
		// A column different from the last one has gone to a new page: move to the next page and draw the row
		r.table.pdf.SetPage(r.child.page)
		r.child.Draw()
		r.table.row = r.child
	} else if r.parent != nil {
		// The last column has gone to a new page: go back to the previous page to draw the row
		r.table.pdf.SetPage(r.parent.page)
		// Remove the relationship to avoid an infinite loop
		r.parent.child = nil
		r.parent.Draw()
		r.table.pdf.SetPage(r.parent.page + 1)
		r.table.pdf.SetY(math.Max(r.highestY(), r.startY+r.height))
	}
}

func (r *Row) Height() float64 {
	if r.height > 0 {
		return r.height
	}
	rh := r.highestY() - r.startY
	if rh > 0 {
		return rh
	}
	return 0
}

func newRow(table *Table) *Row {
	cells := make([]Cell, len(table.columns))
	for i := range cells {
		cells[i] = Cell{}
	}
	var h float64
	if table.row != nil && table.row.height != 0 {
		h = table.row.height
	}
	return &Row{table: table, page: table.pdf.PageNo(), startY: table.pdf.GetY(), cells: cells, height: h}
}

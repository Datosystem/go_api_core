package table

import (
	"strings"

	"github.com/phpdave11/gofpdf"
	"golang.org/x/net/html"
)

type Table struct {
	pdf         *gofpdf.Fpdf
	startX      float64
	endX        float64
	row         *Row
	columns     []Column
	columnIndex int
	headerFunc  func()
	footerFunc  func()
	onPageBreak func()
}

// Calls the tableHeader method if exists
func (t *Table) Headers() map[string]func(*gofpdf.Fpdf) {
	return map[string]func(*gofpdf.Fpdf){
		"tableHeader": func(*gofpdf.Fpdf) {
			if t.headerFunc != nil {
				colIndex := t.columnIndex
				prevRow := t.row
				t.columnIndex = 0
				t.row = newRow(t)
				t.headerFunc()
				t.row = prevRow
				t.columnIndex = colIndex
			}
			curr := t.row
			t.row = newRow(t)
			t.row.parent = curr
		},
	}
}

func (t *Table) Footers() map[string]func(*gofpdf.Fpdf) {
	return map[string]func(*gofpdf.Fpdf){
		"tableFooter": func(*gofpdf.Fpdf) {
			if t.footerFunc != nil {
				t.footerFunc()
			}
		},
	}
}

// Calls the tablePageBreak method of the table if exists by default it keeps printing the table in a new page
func (t *Table) AcceptPageBreak() map[string]func() bool {
	return map[string]func() bool{
		"tablePageBreak": func() bool {
			fontSize, _ := t.pdf.GetFontSize()
			// Update the y-coordinate of the column that has gone to a new page
			t.row.cells[t.columnIndex].endY = t.pdf.GetY()
			if t.onPageBreak != nil {
				t.onPageBreak()
			}
			if t.row.child == nil {
				// If the row does NOT contain the child property, it means the new page has NOT already been created, so create it
				return true
			} else {
				// If the row contains the child property, it means the new page has already been created, so do NOT create it
				t.row = t.row.child
				t.pdf.SetPage(t.row.page)
				t.pdf.SetXY(t.columnX(), t.row.startY)
				t.pdf.SetFontSize(fontSize)
				return false
			}
		},
	}
}

// Used for setting the headerFunc method
func (t *Table) SetHeaderFunc(headerFunc func()) *Table {
	t.headerFunc = headerFunc
	t.headerFunc()
	return t
}

// Used for setting the footerFunc method
func (t *Table) SetFooterFunc(footerFunc func()) *Table {
	t.footerFunc = footerFunc
	return t
}

// SetSizes overrides the current column sizes with the provided ones
// It also initializes the styles array with the columns' size
// This method resets columnIndex and rowHeight to 0 when called
func (t *Table) SetSizes(sizes ...float64) *Table {
	width := t.endX - t.startX
	var relWidth float64
	for _, size := range sizes {
		relWidth += size
	}
	t.columns = make([]Column, len(sizes))
	for i, size := range sizes {
		t.columns[i].size = (width * size) / relWidth
	}
	t.columnIndex = 0
	t.row = newRow(t)
	t.row.cells = make([]Cell, len(t.columns))
	for i := range sizes {
		t.row.cells[i] = Cell{}
	}
	return t
}

// Columns returns the columns of the table
func (t *Table) Columns() *[]Column {
	return &t.columns
}

// Col returns the column by index; returns nil if the column is not found
func (t *Table) Col(index int) *Column {
	if index >= 0 && index < len(t.columns) {
		return &t.columns[index]
	}
	return nil
}

// Sets the style of the different columns
func (t *Table) SetStyles(styles ...*Style) *Table {
	for i := range t.columns {
		if i < len(styles) {
			t.columns[i].setStyle(styles[i])
		} else {
			t.columns[i].setStyle(nil)
		}
	}
	for i := range t.row.cells {
		t.row.cells[i].style = nil
	}
	return t
}

// SetStyle sets the provided style for all columns
func (t *Table) SetStyle(style *Style) *Table {
	styles := make([]*Style, len(t.columns))
	for i := range t.columns {
		styles[i] = style
	}
	return t.SetStyles(styles...)
}

// getStyle returns the provided style or the column style
func (t *Table) getStyle(styles ...*Style) *Style {
	var style Style
	if len(styles) > 0 {
		style = *styles[0]
	} else if t.columns[t.columnIndex].Style != nil {
		style = *t.columns[t.columnIndex].Style
	}
	return &style
}

func (t *Table) RowHeight() float64 {
	return t.row.Height()
}

// Sets the desired row height for the next rows.
//
// Set it to 0 to reset to dynamic height.
func (t *Table) SetRowHeight(rowHeight float64) *Table {
	t.row.height = rowHeight
	return t
}

// Add the specified value and style
func (t *Table) Add(value any, styles ...*Style) *Table {
	t.row.Add(value, styles...)
	t.row.cells[t.columnIndex].endY = t.pdf.GetY()
	return t
}

// AddInline adds the value without changing the Y
func (t *Table) AddInline(value any, styles ...*Style) *Table {
	style := t.getStyle(styles...)
	style.inline = true
	t.Add(value, style)
	return t
}

// Add the specified value, style and
// skips to the next column if there is one otherwise it skips to the next row
func (t *Table) AddNext(value any, styles ...*Style) *Table {
	t.Add(value, styles...)
	t.Next()
	return t
}

func (t *Table) AddImage(imageNameStr string) *Table {
	if t.pdf.GetImageInfo(imageNameStr) != nil {
		if t.RowHeight() > 0 {
			t.pdf.ImageOptions(imageNameStr, t.columnX(), t.pdf.GetY(), t.ColumnSize(), t.RowHeight(), true, gofpdf.ImageOptions{}, 0, "")
		} else {
			t.pdf.ImageOptions(imageNameStr, t.columnX(), t.pdf.GetY(), t.ColumnSize(), 0, true, gofpdf.ImageOptions{}, 0, "")
		}
		t.row.cells[t.columnIndex].endY = t.pdf.GetY()
	}
	return t
}

func (t *Table) AddHTML(htmlStr string, styles ...*Style) *Table {
	reader := strings.NewReader(htmlStr)
	node, err := html.Parse(reader)
	if err != nil {
		t.pdf.SetErrorf("AddHTML error %e", err)
		return t
	}

	columnStyle := *t.getStyle()
	style := columnStyle
	if len(styles) > 0 {
		style = *styles[0]
	}

	element := PdfHTMLElement{
		Width:   t.columns[t.columnIndex].size - columnStyle.PaddingLeft - columnStyle.PaddingRight,
		OffsetX: t.columnX(),
		OffsetY: t.pdf.GetY() + columnStyle.PaddingTop,
		Style:   *style.Inherit(),
	}

	if element.Style.Format == "" {
		element.Style.Format = "-"
	}

	// TODO: Handle padding for child elements
	t.parseHTMLNode(node, &element)

	element.Height += columnStyle.PaddingBottom

	cm := t.pdf.GetCellMargin()
	t.pdf.SetCellMargin(0)

	fRe, fGr, fBl := t.pdf.GetDrawColor()
	tRe, tGr, tBl := t.pdf.GetTextColor()

	var yVariation float64

	t.addHTMLElement(&element, &yVariation)

	t.onPageBreak = nil

	if cm != 0 {
		t.pdf.SetCellMargin(cm)
	}

	t.pdf.SetDrawColor(fRe, fGr, fBl)
	t.pdf.SetTextColor(tRe, tGr, tBl)
	if yVariation == 0 {
		t.pdf.SetXY(element.OffsetX, element.OffsetY+element.Height)
	} else {
		t.pdf.SetX(element.OffsetX)
	}
	t.row.cells[t.columnIndex].endY = t.pdf.GetY()
	return t
}

func (t *Table) addHTMLElement(element *PdfHTMLElement, yVariation *float64) {
	t.onPageBreak = func() {
		*yVariation += element.OffsetY
	}
	if element.Data == "text" {
		if element.Text != "" {
			if len(element.Style.Format) > 0 {
				if element.Style.Format == "-" {
					t.pdf.SetFontStyle("")
				} else {
					t.pdf.SetFontStyle(element.Style.Format)
				}
			}

			fontSize, fontHeight := t.pdf.GetFontSize()

			if element.Style.Size > 0 {
				t.pdf.SetFontSize(element.Style.Size)
			}

			t.pdf.SetXY(element.OffsetX, element.OffsetY)
			t.row.cells[t.columnIndex].endX = element.OffsetX

			t.pdf.CellFormat(element.Width+(t.pdf.GetCellMargin()*2), fontHeight+element.Style.Ln, element.Text, "", 0, "", false, 0, "")

			if element.Style.Size > 0 {
				t.pdf.SetFont("", "", fontSize)
			}
		}
	} else {
		// Store current styles
		fR, fG, fB := t.pdf.GetFillColor()
		dR, dG, dB := t.pdf.GetDrawColor()
		tR, tG, tB := t.pdf.GetTextColor()

		// Apply new styles
		if element.Style.Fill != nil && len(*element.Style.Fill) == 3 {
			fill := *element.Style.Fill
			t.pdf.SetFillColor(fill[0], fill[1], fill[2])

			t.pdf.Rect(element.OffsetX, element.OffsetY, element.Width, element.Height, "F")
			if len(element.Style.Border) == 0 {
				element.Style.Border = "1"
			}
		}
		if element.Style.Color != nil && len(*element.Style.Color) == 3 {
			color := *element.Style.Color
			t.pdf.SetTextColor(color[0], color[1], color[2])
		}

		if len(element.Style.Border) > 0 {
			t.DrawBorder(element.Style.Border, element.OffsetX, element.OffsetY, element.Width, element.Height)
		}

		if element.Style.Fill != nil && len(*element.Style.Fill) == 3 {
			fill := *element.Style.Fill
			t.pdf.SetFillColor(fill[0], fill[1], fill[2])
			t.pdf.SetDrawColor(fill[0], fill[1], fill[2])
		}
		if element.Style.Color != nil && len(*element.Style.Color) == 3 {
			color := *element.Style.Color
			t.pdf.SetTextColor(color[0], color[1], color[2])
		}

		// Add children
		for _, child := range element.Children {
			child.OffsetX += element.OffsetX
			child.OffsetY += element.OffsetY - *yVariation
			t.addHTMLElement(child, yVariation)
		}

		// Reset previous styles
		t.pdf.SetFillColor(fR, fG, fB)
		t.pdf.SetDrawColor(dR, dG, dB)
		t.pdf.SetTextColor(tR, tG, tB)
	}
}

func (t *Table) DrawBorder(borderStr string, x, y, w, h float64) {
	if borderStr == "1" {
		borderStr = "TRBL"
	}
	if strings.Contains(borderStr, "T") {
		t.pdf.Line(x, y, x+w, y)
	}
	if strings.Contains(borderStr, "R") {
		t.pdf.Line(x+w, y, x+w, y+h)
	}
	if strings.Contains(borderStr, "B") {
		t.pdf.Line(x, y+h, x+w, y+h)
	}
	if strings.Contains(borderStr, "L") {
		t.pdf.Line(x, y, x, y+h)
	}
}

// Skips to the next column if there is one otherwise it skips to the next row
func (t *Table) Next() *Table {
	currentCell := t.row.cells[t.columnIndex]
	if currentCell.style != nil && currentCell.style.inline {
		_, fontHeight := t.pdf.GetFontSize()
		t.pdf.Ln(fontHeight + currentCell.style.Ln)
		t.row.cells[t.columnIndex].endY = t.pdf.GetY()
	}
	t.columnIndex++
	if t.columnIndex >= len(t.columns) {
		t.nextRow()
	} else if t.row.parent != nil {
		t.row.parent.child = t.row
		t.row = t.row.parent
		t.pdf.SetPage(t.row.page)
	}
	t.pdf.SetXY(t.columnX(), t.row.startY)
	return t
}

// Prints the current table and then draws a new row
func (t *Table) NextRow() *Table {
	l := len(t.columns) - t.columnIndex
	for i := 0; i < l; i++ {
		t.Next()
	}
	return t
}

// Prints the current table and then draws a new row
func (t *Table) nextRow() {
	t.row.Draw()
	t.columnIndex = 0
	t.row = newRow(t)
}

// AddMany adds multiple columns to the table
func (t *Table) AddMany(values ...any) *Table {
	for _, val := range values {
		t.AddNext(val)
	}
	return t
}

func (t *Table) originalColumnX(index int) float64 {
	x := t.startX
	for i := 0; i < index; i++ {
		x += t.columns[i].size
	}
	return x
}

func (t *Table) columnX(colIndex ...int) float64 {
	index := t.columnIndex
	if len(colIndex) > 0 {
		index = colIndex[0]
	}
	x := t.originalColumnX(index)
	if t.columns[index].Style != nil {
		x += t.columns[index].Style.PaddingLeft
	}
	return x
}

/*func (t *Table) RowY() float64 {
	return t.row.startY
}*/

func (t *Table) originalColumnSize(index int) float64 {
	return t.columns[index].size
}

func (t *Table) ColumnSize(colIndex ...int) float64 {
	index := t.columnIndex
	if len(colIndex) > 0 {
		index = colIndex[0]
	}
	size := t.originalColumnSize(index)
	if t.columns[index].Style != nil {
		size -= t.columns[index].Style.PaddingLeft + t.columns[index].Style.PaddingRight
	}
	return size
}

func (t *Table) ColumnBoundaries(colIndex ...int) (x1 float64, x2 float64) {
	index := t.columnIndex
	if len(colIndex) > 0 {
		index = colIndex[0]
	}
	x1 = t.columnX(index)
	x2 = x1 + t.ColumnSize(index)
	return
}

// Prints the table
func (t *Table) Close() *Table {
	t.row.Draw()
	return t
}

// Hr will draw an horizontal line at the bottom of the current row's next Draw call
func (t *Table) Hr() *Table {
	t.row.hr = true
	return t
}

func New(p *gofpdf.Fpdf, x1 float64, x2 float64) *Table {
	return &Table{pdf: p, startX: x1, endX: x2}
}

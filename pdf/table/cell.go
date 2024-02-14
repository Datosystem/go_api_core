package table

type Cell struct {
	style *Style
	endX  float64
	endY  float64
}

func (c Cell) Style(column *Column) *Style {
	if c.style != nil {
		return c.style
	}
	if column.Style != nil {
		return column.Style
	}
	return nil
}

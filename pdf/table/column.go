package table

type Column struct {
	Style *Style
	size  float64
}

func (c *Column) setStyle(style *Style) {
	if style == nil {
		c.Style = nil
	} else {
		c.Style = style.Clone()
	}
}

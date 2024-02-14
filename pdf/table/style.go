package table

type Style struct {
	VAlign        string
	Align         string
	Border        string
	Format        string
	Size          float64
	Ln            float64
	Fill          *[]int
	Color         *[]int
	PaddingTop    float64
	PaddingRight  float64
	PaddingBottom float64
	PaddingLeft   float64
	MarginTop     float64
	MarginRight   float64
	MarginBottom  float64
	MarginLeft    float64
	inline        bool
}

// Clone the specified style
func (s *Style) Clone() *Style {
	clone := *s
	return &clone
}

func (s *Style) Inherit() *Style {
	child := Style{}
	child.Format = s.Format
	child.Size = s.Size
	child.Ln = s.Ln
	if s.Fill != nil {
		fill := *s.Fill
		child.Fill = &fill
	}
	if s.Color != nil {
		color := *s.Color
		child.Color = &color
	}
	return &child
}

// Assign multiple styles (basically, it picks the one initialized and NOT at the empty value)
func (s *Style) Assign(styles ...*Style) *Style {
	dest := s.Clone()
	for _, style := range styles {
		if style.VAlign != "" {
			dest.VAlign = style.VAlign
		}
		if style.Align != "" {
			dest.Align = style.Align
		}
		if style.Border != "" {
			dest.Border = style.Border
		}
		if style.Format != "" {
			dest.Format = style.Format
		}
		if style.Size > 0 {
			dest.Size = style.Size
		}
		if style.Ln > 0 {
			dest.Ln = style.Ln
		}
		if style.Fill != nil {
			dest.Fill = style.Fill
		}
		if style.Color != nil {
			dest.Color = style.Color
		}
		if style.PaddingTop != 0 {
			dest.PaddingTop = style.PaddingTop
		}
		if style.PaddingRight != 0 {
			dest.PaddingRight = style.PaddingRight
		}
		if style.PaddingBottom != 0 {
			dest.PaddingBottom = style.PaddingBottom
		}
		if style.PaddingLeft != 0 {
			dest.PaddingLeft = style.PaddingLeft
		}
		if style.MarginTop != 0 {
			dest.MarginTop = style.MarginTop
		}
		if style.MarginRight != 0 {
			dest.MarginRight = style.MarginRight
		}
		if style.MarginBottom != 0 {
			dest.MarginBottom = style.MarginBottom
		}
		if style.MarginLeft != 0 {
			dest.MarginLeft = style.MarginLeft
		}
		dest.inline = style.inline
	}
	return dest
}

func (s *Style) Padding(padding ...float64) *Style {
	if len(padding) > 0 {
		switch len(padding) {
		case 1:
			{
				s.PaddingTop = padding[0]
				s.PaddingRight = padding[0]
				s.PaddingBottom = padding[0]
				s.PaddingLeft = padding[0]
			}
		case 2:
			{
				s.PaddingTop = padding[0]
				s.PaddingRight = padding[1]
				s.PaddingBottom = padding[0]
				s.PaddingLeft = padding[1]
			}
		case 4:
			{
				s.PaddingTop = padding[0]
				s.PaddingRight = padding[1]
				s.PaddingBottom = padding[2]
				s.PaddingLeft = padding[3]
			}
		}
	}
	return s
}

func (s *Style) Margin(margin ...float64) *Style {
	if len(margin) > 0 {
		switch len(margin) {
		case 1:
			{
				s.MarginTop = margin[0]
				s.MarginRight = margin[0]
				s.MarginBottom = margin[0]
				s.MarginLeft = margin[0]
			}
		case 2:
			{
				s.MarginTop = margin[0]
				s.MarginRight = margin[1]
				s.MarginBottom = margin[0]
				s.MarginLeft = margin[1]
			}
		case 4:
			{
				s.MarginTop = margin[0]
				s.MarginRight = margin[1]
				s.MarginBottom = margin[2]
				s.MarginLeft = margin[3]
			}
		}
	}
	return s
}

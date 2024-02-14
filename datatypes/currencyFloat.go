package datatypes

import "math"

type CurrencyFloat float32

func (c *CurrencyFloat) Scan(value interface{}) (err error) {
	f := value.(float64)
	*c = CurrencyFloat(math.Round(f*100) / 100)
	return
}

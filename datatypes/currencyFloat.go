package datatypes

import (
	"fmt"
)

type CurrencyFloat string

func (c *CurrencyFloat) Scan(value interface{}) (err error) {
	*c = CurrencyFloat(fmt.Sprintf("%.2f", value))
	return
}

package datatypes

import (
	"database/sql/driver"
	"math"
)

type RoundedFloat float32

func (f RoundedFloat) Value() (driver.Value, error) {
	return math.Round(float64(f*10000)) / 10000, nil
}

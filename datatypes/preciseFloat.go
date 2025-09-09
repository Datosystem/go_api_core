package datatypes

import (
	"database/sql/driver"
	"math"
)

type PreciseFloat float32

func (f PreciseFloat) Value() (driver.Value, error) {
	return math.Round(float64(f*100000000)) / 100000000, nil
}

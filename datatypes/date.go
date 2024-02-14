package datatypes

import (
	"database/sql"
	"database/sql/driver"
	"time"
)

type Date time.Time

func (date *Date) Scan(value interface{}) (err error) {
	nullTime := &sql.NullTime{}
	err = nullTime.Scan(value)
	if err == nil {
		*date = Date(nullTime.Time)
	} else if s, ok := value.(string); ok {
		t, _ := time.Parse(time.RFC3339, s)
		*date = Date(t)
	}
	return
}

func (date Date) String() string {
	y, m, d := time.Time(date).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Time(date).Location()).Local().String()
}

func (date Date) Value() (driver.Value, error) {
	y, m, d := time.Time(date).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Time(date).Location()), nil
}

// GormDataType gorm common data type
func (date Date) GormDataType() string {
	return "date"
}

func (date Date) GobEncode() ([]byte, error) {
	return time.Time(date).GobEncode()
}

func (date *Date) GobDecode(b []byte) error {
	return (*time.Time)(date).GobDecode(b)
}

func (date Date) MarshalJSON() ([]byte, error) {
	return time.Time(date).MarshalJSON()
}

func (date *Date) UnmarshalJSON(b []byte) error {
	return (*time.Time)(date).UnmarshalJSON(b)
}

package datatypes

import (
	"database/sql"
	"database/sql/driver"
	"time"
)

type Datetime time.Time

func (date *Datetime) Scan(value interface{}) (err error) {
	nullTime := &sql.NullTime{}
	err = nullTime.Scan(value)
	if err == nil {
		*date = Datetime(nullTime.Time)
	} else if s, ok := value.(string); ok {
		t, _ := time.Parse(time.RFC3339, s)
		*date = Datetime(t)
	}
	return
}

func (date Datetime) Value() (driver.Value, error) {
	t := time.Time(date)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location()), nil
}

// GormDataType gorm common data type
func (date Datetime) GormDataType() string {
	return "datetime"
}

func (date Datetime) GobEncode() ([]byte, error) {
	return time.Time(date).GobEncode()
}

func (date *Datetime) GobDecode(b []byte) error {
	return (*time.Time)(date).GobDecode(b)
}

func (date Datetime) MarshalJSON() ([]byte, error) {
	return time.Time(date).MarshalJSON()
}

func (date *Datetime) UnmarshalJSON(b []byte) error {
	return (*time.Time)(date).UnmarshalJSON(b)
}

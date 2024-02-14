package controller

import (
	"strconv"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ShouldPaginate(pagStart, pagEnd string) bool {
	return len(pagStart) > 0 || len(pagEnd) > 0
}

func GetOffset(pagStart string) int {
	offset, err := strconv.Atoi(pagStart)
	if err != nil {
		return 0
	}
	return offset
}

func GetLimit(pagStart, pagEnd string) int {
	end, err := strconv.Atoi(pagEnd)
	if err != nil {
		return 100
	}
	return end - GetOffset(pagStart)
}

func Count(count *int64) func(*gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		if d.Statement.Distinct {
			n := clause.OrderBy{}.Name()
			ord := d.Statement.Clauses[n]
			delete(d.Statement.Clauses, n)
			d.Session(&gorm.Session{}).Raw("SELECT COUNT(1) FROM (?) AS T", d).Scan(count)
			d.Statement.Clauses[n] = ord
		} else {
			d.Session(&gorm.Session{}).Count(count)
		}
		return d
	}
}

func Paginate(pagStart, pagEnd string) func(*gorm.DB) *gorm.DB {
	if ShouldPaginate(pagStart, pagEnd) {
		offset := GetOffset(pagStart)
		limit := GetLimit(pagStart, pagEnd)

		return func(query *gorm.DB) *gorm.DB {
			return query.Offset(offset).Limit(limit)
		}
	} else {
		return func(query *gorm.DB) *gorm.DB {
			return query
		}
	}
}

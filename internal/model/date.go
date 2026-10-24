package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Date 以 "YYYY-MM-DD" 字符串与 MySQL DATE 列互转，避免时区漂移。
type Date string

// Scan 实现 sql.Scanner。
func (d *Date) Scan(v any) error {
	switch t := v.(type) {
	case nil:
		*d = ""
	case time.Time:
		*d = Date(t.Format("2006-01-02"))
	case []byte:
		*d = Date(string(t))
	case string:
		if len(t) >= 10 {
			*d = Date(t[:10])
		} else {
			*d = Date(t)
		}
	default:
		return fmt.Errorf("model.Date: unsupported scan type %T", v)
	}
	return nil
}

// Value 实现 driver.Valuer。
func (d Date) Value() (driver.Value, error) {
	if d == "" {
		return nil, nil
	}
	return string(d), nil
}

// Today 返回 UTC 当日日期。
func Today() Date { return Date(time.Now().UTC().Format("2006-01-02")) }

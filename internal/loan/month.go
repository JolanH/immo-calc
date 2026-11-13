package loan

import (
	"fmt"
	"regexp"
	"strconv"
)

// Month is a calendar month, without a day. It reads and writes as "YYYY-MM"
// in JSON and YAML.
type Month struct {
	Year  int
	Month int // 1–12
}

var monthPattern = regexp.MustCompile(`^(\d{4})-(\d{2})$`)

// ParseMonth parses "YYYY-MM".
func ParseMonth(s string) (Month, error) {
	m := monthPattern.FindStringSubmatch(s)
	if m == nil {
		return Month{}, fmt.Errorf("parse month %q: want YYYY-MM", s)
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	if month < 1 || month > 12 {
		return Month{}, fmt.Errorf("parse month %q: month out of range", s)
	}
	return Month{Year: year, Month: month}, nil
}

// IsZero reports whether m is unset.
func (m Month) IsZero() bool { return m == Month{} }

// Add returns the month n months after m (n may be negative).
func (m Month) Add(n int) Month {
	total := m.Year*12 + (m.Month - 1) + n
	return Month{Year: total / 12, Month: total%12 + 1}
}

// Next returns the following month.
func (m Month) Next() Month { return m.Add(1) }

// String formats m as "YYYY-MM".
func (m Month) String() string { return fmt.Sprintf("%04d-%02d", m.Year, m.Month) }

// MarshalText implements encoding.TextMarshaler (JSON and YAML).
func (m Month) MarshalText() ([]byte, error) { return []byte(m.String()), nil }

// UnmarshalText implements encoding.TextUnmarshaler (JSON and YAML).
func (m *Month) UnmarshalText(text []byte) error {
	parsed, err := ParseMonth(string(text))
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

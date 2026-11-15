package expr

import "time"

// Context provides the injected clock, location, and configuration for evaluation.
type Context struct {
	Now time.Time
	Loc *time.Location
	Cfg Config
}

// Config holds configurable evaluation rules.
type Config struct {
	Lang           Lang           // en
	Strict         bool           // false
	WeekStart      time.Weekday   // time.Monday
	WeekOne        string         // "iso" | "first-day" | "full" | "days"
	MonthOverflow  bool           // false = clamp to the month's last day
	CountInclusive bool           // false
	Weekend        []time.Weekday // [time.Saturday, time.Sunday]
	Country        string         // "MX"; "" disables public holidays
	DaysOff        []Value        // additional non-business dates or whole-day spans
}

// DefaultConfig returns the default evaluation configuration.
func DefaultConfig() Config {
	return Config{
		Lang:      EN,
		WeekStart: time.Monday,
		WeekOne:   "iso",
		Weekend:   []time.Weekday{time.Saturday, time.Sunday},
		Country:   "MX",
	}
}

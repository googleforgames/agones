package iso8601

import (
	"testing"
)

type TestCase struct {
	Using string

	Year  int
	Month int
	Day   int

	Hour        int
	Minute      int
	Second      int
	MilliSecond int

	Zone            float64
	ShouldFailParse bool

	ShouldInvalidRange      bool
	RangeElementWhenInvalid string
}

func (tc TestCase) CheckError(err error, t *testing.T) bool {
	if err != nil {
		if tc.ShouldInvalidRange {
			re, ok := err.(*RangeError)
			if !ok {
				t.Fatalf("Found error %s of type %T but was expecting a RangeError", err, err)
			}

			if tc.RangeElementWhenInvalid != "" && re.Element != tc.RangeElementWhenInvalid {
				t.Fatalf("Expected a range error on %q but encountered %q: %s", tc.RangeElementWhenInvalid, re.Element, err)
			}

			return true
		}
		if tc.ShouldFailParse {
			return true
		}

		t.Fatal(err)
		return false
	}

	if err == nil && (tc.ShouldFailParse || tc.ShouldInvalidRange) {
		reason := "fail to parse"
		if tc.ShouldInvalidRange {
			reason = "to catch an invalid date range"
		}
		t.Fatalf("Expected test case %s", reason)
		return true
	}

	return false
}

var cases = []TestCase{
	{
		Using: "2017-04-24T09:41:34.502+0100",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        1,
	},
	{
		Using: "2017-04-24T09:41+0100",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41,
		Zone: 1,
	},
	{
		Using: "2017-04-24T09+0100",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9,
		Zone: 1,
	},
	{
		Using: "2017-04-24T",
		Year:  2017, Month: 4, Day: 24,
	},
	{
		Using: "2017-04-24",
		Year:  2017, Month: 4, Day: 24,
	},
	{
		Using: "2017-04-24T09:41:34+0100",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		Zone: 1,
	},
	{
		Using: "2017-04-24T09:41:34.502-0100",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        -1,
	},
	{
		Using: "2017-04-24T09:41:34.502-01:00",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        -1,
	},
	{
		Using: "2017-04-24T09:41-01:00",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41,
		Zone: -1,
	},
	{
		Using: "2017-04-24T09-01:00",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9,
		Zone: -1,
	},
	{
		Using: "2017-04-24T09:41:34-0100",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		Zone: -1,
	},
	{
		Using: "2017-04-24T09:41:34.502Z",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        0,
	},
	{
		Using: "2017-04-24T09:41:34Z",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		Zone: 0,
	},
	{
		Using: "2017-04-24T09:41Z",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41,
		Zone: 0,
	},
	{
		Using: "2017-04-24T09Z",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9,
		Zone: 0,
	},
	{
		Using: "2017-04-24T09:41:34.089",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 89,
		Zone:        0,
	},
	{
		Using: "2017-04-24T09:41",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41,
		Zone: 0,
	},
	{
		Using: "2017-04-24T09",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9,
		Zone: 0,
	},
	{
		Using: "2017-04-24T09:41:34.009",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 9,
		Zone:        0,
	},
	{
		Using: "2017-04-24T09:41:34.893",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 893,
		Zone:        0,
	},
	{
		Using: "2017-04-24T09:41:34.89312523Z",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 893,
		Zone:        0,
	},
	{
		Using: "2017-04-24T09:41:34.502-0530",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        -5.5,
	},
	{
		Using: "2017-04-24T09:41:34.502+0530",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        5.5,
	},
	{
		Using: "2017-04-24T09:41:34.502+05:30",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        5.5,
	},

	{
		Using: "2017-04-24T09:41:34.502+05:45",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        5.75,
	},
	{
		Using: "2017-04-24T09:41:34.502+00",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        0,
	},
	{
		Using: "2017-04-24T09:41:34.502+0000",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        0,
	},
	{
		Using: "2017-04-24T09:41:34.502+00:00",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        0,
	},
	{
		Using: "+2017-04-24T09:41:34.502+00:00",
		Year:  2017, Month: 4, Day: 24,
		Hour: 9, Minute: 41, Second: 34,
		MilliSecond: 502,
		Zone:        0,
	},
	{
		Using: "2017",
		Year:  2017, Month: 1, Day: 1, Hour: 0, Minute: 0, Second: 0,
		MilliSecond: 0,
		Zone:        0,
	},

	// Invalid Parse Test Cases
	{
		Using:           "2017-04-24T09:41:34.502-00",
		ShouldFailParse: true,
	},
	{
		Using:           "2017-04-24T09:41:34.502-0000",
		ShouldFailParse: true,
	},
	{
		Using:           "2017-04-24T09:41:34.502-00:00",
		ShouldFailParse: true,
	},
	{
		Using:           "-2017-04-24T09:41:34.502-00:00",
		ShouldFailParse: true,
	},
	{
		Using:           "2017-+04-24T09:41:34.502-00:00",
		ShouldFailParse: true,
	},

	// Invalid Range Test Cases
	{
		Using:                   "2017-00-01T00:00:00.000+00:00",
		ShouldInvalidRange:      true,
		RangeElementWhenInvalid: "month",
	},
	{
		Using:                   "2017-13-01T00:00:00.000+00:00",
		ShouldInvalidRange:      true,
		RangeElementWhenInvalid: "month",
	},

	{
		Using:                   "2017-01-00T00:00:00.000+00:00",
		ShouldInvalidRange:      true,
		RangeElementWhenInvalid: "day",
	},
	{
		Using:                   "2017-01-32T00:00:00.000+00:00",
		ShouldInvalidRange:      true,
		RangeElementWhenInvalid: "day",
	},
	{
		Using:                   "2019-02-29T00:00:00.000+00:00",
		ShouldInvalidRange:      true,
		RangeElementWhenInvalid: "day",
	},
	{
		Using:                   "2020-02-30T00:00:00.000+00:00", // Leap year
		ShouldInvalidRange:      true,
		RangeElementWhenInvalid: "day",
	},

	{
		Using:                   "2017-01-01T24:00:00.000+00:00",
		ShouldInvalidRange:      true,
		RangeElementWhenInvalid: "hour",
	},

	{
		Using:                   "2017-01-01T00:60:00.000+00:00",
		ShouldInvalidRange:      true,
		RangeElementWhenInvalid: "minute",
	},

	{
		Using:                   "2017-01-01T00:00:60.000+00:00",
		ShouldInvalidRange:      true,
		RangeElementWhenInvalid: "second",
	},
}

func TestParse(t *testing.T) {
	for _, c := range cases {
		t.Run(c.Using, func(t *testing.T) {
			d, err := Parse([]byte(c.Using))
			if c.CheckError(err, t) {
				return
			}
			t.Log(d)

			if y := d.Year(); y != c.Year {
				t.Errorf("Year = %d; want %d", y, c.Year)
			}
			if m := int(d.Month()); m != c.Month {
				t.Errorf("Month = %d; want %d", m, c.Month)
			}
			if d := d.Day(); d != c.Day {
				t.Errorf("Day = %d; want %d", d, c.Day)
			}
			if h := d.Hour(); h != c.Hour {
				t.Errorf("Hour = %d; want %d", h, c.Hour)
			}
			if m := d.Minute(); m != c.Minute {
				t.Errorf("Minute = %d; want %d", m, c.Minute)
			}
			if s := d.Second(); s != c.Second {
				t.Errorf("Second = %d; want %d", s, c.Second)
			}

			if ms := d.Nanosecond() / 1000000; ms != c.MilliSecond {
				t.Errorf("Millisecond = %d; want %d (%d nanoseconds)", ms, c.MilliSecond, d.Nanosecond())
			}

			_, z := d.Zone()
			if offset := float64(z) / 3600; offset != c.Zone {
				t.Errorf("Zone = %.2f (%d); want %.2f", offset, z, c.Zone)
			}
		})

	}
}

func TestParseString(t *testing.T) {
	for _, c := range cases {
		t.Run(c.Using, func(t *testing.T) {
			d, err := ParseString(c.Using)
			if c.CheckError(err, t) {
				return
			}
			t.Log(d)

			if y := d.Year(); y != c.Year {
				t.Errorf("Year = %d; want %d", y, c.Year)
			}
			if m := int(d.Month()); m != c.Month {
				t.Errorf("Month = %d; want %d", m, c.Month)
			}
			if d := d.Day(); d != c.Day {
				t.Errorf("Day = %d; want %d", d, c.Day)
			}
			if h := d.Hour(); h != c.Hour {
				t.Errorf("Hour = %d; want %d", h, c.Hour)
			}
			if m := d.Minute(); m != c.Minute {
				t.Errorf("Minute = %d; want %d", m, c.Minute)
			}
			if s := d.Second(); s != c.Second {
				t.Errorf("Second = %d; want %d", s, c.Second)
			}

			if ms := d.Nanosecond() / 1000000; ms != c.MilliSecond {
				t.Errorf("Millisecond = %d; want %d (%d nanoseconds)", ms, c.MilliSecond, d.Nanosecond())
			}

			_, z := d.Zone()
			if offset := float64(z) / 3600; offset != c.Zone {
				t.Errorf("Zone = %.2f (%d); want %.2f", offset, z, c.Zone)
			}
		})

	}
}

func BenchmarkParse(b *testing.B) {
	x := []byte("2017-04-24T09:41:34.502Z")
	for i := 0; i < b.N; i++ {
		_, err := Parse(x)
		if err != nil {
			b.Fatal(err)
		}
	}
}

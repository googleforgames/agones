package iso8601

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

type TestAPIResponse struct {
	Ptr  *Time
	Nptr Time
}

type TestStdLibAPIResponse struct {
	Ptr  *time.Time
	Nptr time.Time
}

var ShortTest = TestCase{
	Using: "2001-11-13",
	Year:  2001, Month: 11, Day: 13,
}

var StructTestData = []byte(`
{
  "Ptr": "2017-04-26T11:13:04+01:00",
  "Nptr": "2017-04-26T11:13:04+01:00"
}
`)

var NullTestData = []byte(`
{
  "Ptr": null,
  "Nptr": null
}
`)

var ZeroedTestData = []byte(`
{
  "Ptr": "0001-01-01",
  "Nptr": "0001-01-01"
}
`)

var StructTest = TestCase{
	Year: 2017, Month: 04, Day: 26,
	Hour: 11, Minute: 13, Second: 04,
	Zone: 1,
}

func TestTime_UnmarshalJSON(t *testing.T) {
	t.Run("short", func(t *testing.T) {
		var b = []byte(`"2001-11-13"`)

		tn := new(Time)
		if err := tn.UnmarshalJSON(b); err != nil {
			t.Fatal(err)
		}

		if y := tn.Year(); y != ShortTest.Year {
			t.Errorf("Year = %d; want %d", y, ShortTest.Year)
		}

		if m := int(tn.Month()); m != ShortTest.Month {
			t.Errorf("Month = %d; want %d", m, ShortTest.Month)
		}

		if d := tn.Day(); d != ShortTest.Day {
			t.Errorf("Day = %d; want %d", d, ShortTest.Day)
		}

		err := tn.UnmarshalJSON([]byte(`2001-11-13`))
		if err != ErrNotString {
			t.Fatal(err)
		}
		if err == nil {
			t.Fatal("Expected an error from unmarshal")
		}
	})

	t.Run("struct", func(t *testing.T) {
		resp := new(TestAPIResponse)
		if err := json.Unmarshal(StructTestData, resp); err != nil {
			t.Fatal(err)
		}

		stdlibResp := new(TestStdLibAPIResponse)
		if err := json.Unmarshal(StructTestData, stdlibResp); err != nil {
			t.Fatal(err)
		}

		t.Run("stblib parity", func(t *testing.T) {
			if !resp.Ptr.Equal(*stdlibResp.Ptr) || !resp.Nptr.Equal(stdlibResp.Nptr) {
				t.Fatalf("Parsed time values are not equal to standard library implementation")
			}
		})

		t.Run("ptr", func(t *testing.T) {
			if y := resp.Ptr.Year(); y != StructTest.Year {
				t.Errorf("Ptr: Year = %d; want %d", y, StructTest.Year)
			}
			if d := resp.Ptr.Day(); d != StructTest.Day {
				t.Errorf("Ptr: Day = %d; want %d", d, StructTest.Day)
			}
			if s := resp.Ptr.Second(); s != StructTest.Second {
				t.Errorf("Ptr: Second = %d; want %d", s, StructTest.Second)
			}
		})

		t.Run("noptr", func(t *testing.T) {
			if y := resp.Nptr.Year(); y != StructTest.Year {
				t.Errorf("NoPtr: Year = %d; want %d", y, StructTest.Year)
			}
			if d := resp.Nptr.Day(); d != StructTest.Day {
				t.Errorf("NoPtr: Day = %d; want %d", d, StructTest.Day)
			}
			if s := resp.Nptr.Second(); s != StructTest.Second {
				t.Errorf("NoPtr: Second = %d; want %d", s, StructTest.Second)
			}
		})
	})

	t.Run("null", func(t *testing.T) {
		resp := new(TestAPIResponse)
		if err := json.Unmarshal(NullTestData, resp); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("time zeroed", func(t *testing.T) {
		resp := new(TestAPIResponse)
		if err := json.Unmarshal(ZeroedTestData, resp); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("reparse", func(t *testing.T) {
		s := time.Now().UTC()
		data := []byte(s.Format(time.RFC3339Nano))
		n, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if !s.Equal(n) {
			t.Fatalf("Parsing a JSON date mismatch; wanted %s; got %s", s, n)
		}
	})
}

func BenchmarkCheckNull(b *testing.B) {
	var n = []byte("null")

	b.Run("compare", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bytes.Compare(n, n)
		}
	})
	b.Run("exact", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			null(n)
		}
	})
}

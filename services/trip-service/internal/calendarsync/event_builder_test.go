package calendarsync

import "testing"

func TestParseTimeRangeSupportedFormats(t *testing.T) {
	cases := []string{"09:00", "9:00", "09:00 AM", "9:00 AM", "2:30 PM", "14:30", "09:00-10:30", "09:00 – 10:30"}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			if _, _, ok := parseTimeRange(tc); !ok {
				t.Fatalf("expected %q to parse", tc)
			}
		})
	}
}

func TestParseTimeRangeRejectsUnparseable(t *testing.T) {
	if _, _, ok := parseTimeRange("morning"); ok {
		t.Fatal("expected morning to be skipped")
	}
}

func TestDefaultDuration(t *testing.T) {
	if defaultDuration("food").Minutes() != 90 {
		t.Fatal("food should default to 90 minutes")
	}
	if defaultDuration("transport").Minutes() != 30 {
		t.Fatal("transport should default to 30 minutes")
	}
	if defaultDuration("activity").Minutes() != 60 {
		t.Fatal("activity should default to 60 minutes")
	}
}

// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import "time"

// pointTime extracts a time.Time from a DataPoint.X value, accepting
// time.Time, string (RFC 3339), or numeric milliseconds since epoch.
// The boolean result reports whether a time was successfully extracted.
func pointTime(x any) (time.Time, bool) {
	switch v := x.(type) {
	case time.Time:
		return v, true
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t, true
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, true
		}
	case float64:
		return time.UnixMilli(int64(v)), true
	case int64:
		return time.UnixMilli(v), true
	case int:
		return time.UnixMilli(int64(v)), true
	}
	return time.Time{}, false
}

// pointFloat coerces a DataPoint.X value into a float64 when the X axis is
// numeric ("value"). Returns false if X cannot be represented as a float64.
func pointFloat(x any) (float64, bool) {
	switch v := x.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}

// pointString coerces a DataPoint.X value into a string (category axis).
func pointString(x any) (string, bool) {
	if s, ok := x.(string); ok {
		return s, true
	}
	return "", false
}

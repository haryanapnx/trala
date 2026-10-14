package handler

import "strconv"

// parseQueryInt reads an optional positive integer query parameter, falling back
// to def when it is missing, malformed or not positive. Range clamping (e.g.
// page size) is left to the use case.
func parseQueryInt(raw string, def int32) int32 {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return def
	}
	return int32(v)
}

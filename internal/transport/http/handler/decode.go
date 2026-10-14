package handler

import (
	"encoding/json"
	"io"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// decodeJSON reads a JSON request body into dst, bounding its size.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
	return dec.Decode(dst)
}

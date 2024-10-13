package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func (c Controller) ApiKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var requestBody struct {
			APIKey string `json:"api_key"`
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Restore the request body so it can be read by the next handler
		req.Body = io.NopCloser(io.Reader(bytes.NewBuffer(body)))

		if err := json.Unmarshal(body, &requestBody); err != nil {
			http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
			return
		}

		if requestBody.APIKey != c.apiKey {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, req)
	}
}

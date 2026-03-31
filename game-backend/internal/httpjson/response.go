package httpjson

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func Write(w http.ResponseWriter, status int, payload any) error {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err := body.WriteTo(w)
	return err
}

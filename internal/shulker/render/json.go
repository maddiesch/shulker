package render

import (
	"encoding/json"
	"net/http"
)

func JSON(w http.ResponseWriter, status int, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	w.WriteHeader(status)
	_, err = w.Write(body)

	return err
}

func JSONError(w http.ResponseWriter, status int, message string) error {
	return JSON(w, status, map[string]any{
		"error": map[string]any{
			"message": message,
		},
	})
}

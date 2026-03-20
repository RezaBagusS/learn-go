package dto

import (
	"encoding/json"
	"net/http"
)

type BaseResponse struct {
	Message string         `json:"message,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

func WriteResponse(w http.ResponseWriter, code int, desc string, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(BaseResponse{
		Message: desc,
		Data:    payload,
	})
}

func WriteError(w http.ResponseWriter, code int, desc string) {
	WriteResponse(w, code, desc, nil)
}

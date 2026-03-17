package dto

import (
	"encoding/json"
	"net/http"
)

type BaseResponse struct {
	ResponseCode int            `json:"ResponseCode,omitempty"`
	ResponseDesc string         `json:"ResponseDesc,omitempty"`
	Payload      map[string]any `json:"Payload,omitempty"`
}

func WriteResponse(w http.ResponseWriter, code int, desc string, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(BaseResponse{
		ResponseCode: code,
		ResponseDesc: desc,
		Payload:      payload,
	})
}

func WriteError(w http.ResponseWriter, code int, desc string) {
	WriteResponse(w, code, desc, nil)
}

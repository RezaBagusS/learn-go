package dto

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type BaseResponse struct {
	ResponseCode    string `json:"responseCode,omitempty"`
	ResponseMessage string `json:"responMessage,omitempty"`
	Data            any    `json:"data,omitempty"`
}

func WriteResponse(w http.ResponseWriter, httpStatus int, respCode string, respMsg string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-TIMESTAMP", time.Now().Format(time.RFC3339))
	w.WriteHeader(httpStatus)

	dataBytes, err := json.Marshal(data)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"responseCode":    respCode,
			"responseMessage": respMsg,
		})
		return
	}

	var buf bytes.Buffer

	var dataMap map[string]json.RawMessage
	hasExtra := json.Unmarshal(dataBytes, &dataMap) == nil && len(dataMap) > 0

	if hasExtra {
		delete(dataMap, "responseCode")
		delete(dataMap, "responseMessage")
	}

	buf.WriteString(`{`)
	buf.WriteString(`"responseCode":`)
	codeBytes, _ := json.Marshal(respCode)
	buf.Write(codeBytes)
	buf.WriteString(`,"responseMessage":`)
	msgBytes, _ := json.Marshal(respMsg)
	buf.Write(msgBytes)

	if hasExtra {
		for k, v := range dataMap {
			buf.WriteString(`,`)
			keyBytes, _ := json.Marshal(k)
			buf.Write(keyBytes)
			buf.WriteString(`:`)
			buf.Write(v)
		}
	}

	buf.WriteString(`}`)
	buf.WriteByte('\n')

	w.Write(buf.Bytes())
}

func WriteError(w http.ResponseWriter, httpStatus int, respCode string, respMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	json.NewEncoder(w).Encode(map[string]string{
		"responseCode":    respCode,
		"responseMessage": respMsg,
	})
}

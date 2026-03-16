package dto

type BaseResponse struct {
	ResponseCode string `json:"responseCode,omitempty"`
	ResponseDesc string `json:"responseDesc,omitempty"`
}

package domain

import (
	"fmt"
	"net/http"
)

type SNAPHeader struct {
	Timestamp  string
	Signature  string
	Origin     string
	PartnerID  string
	ExternalID string
	IPAddress  string
	DeviceID   string
	Latitude   string
	Longitude  string
	ChannelID  string
}

type SnapDetail struct {
	Category        string
	HttpCode        int
	CaseCode        string
	ResponseMessage string
	Description     string
	IsAdditional    bool
}

type AdditionalInfo struct {
	DeviceId string `json:"deviceId"`
	Channel  string `json:"channel"`
}

type OriginatorInfo struct {
	OriginatorCustomerNo   string `json:"originator_customer_no"`
	OriginatorCustomerName string `json:"originator_customer_name"`
	OriginatorBankCode     string `json:"originator_bank_code"`
}

type TransactionAdditionalInfo struct {
	DeviceID         string           `json:"deviceId,omitempty"`
	Channel          string           `json:"channel,omitempty"`
	BeneficiaryEmail string           `json:"beneficiaryEmail,omitempty"`
	FeeType          string           `json:"feeType,omitempty"`
	OriginatorInfos  []OriginatorInfo `json:"originatorInfos,omitempty"`
}

func (d SnapDetail) GetResponseCode(ServiceCode string) string {
	return fmt.Sprintf("%d%s%s", d.HttpCode, ServiceCode, d.CaseCode)
}

func ExtractSNAPHeader(r *http.Request) SNAPHeader {
	return SNAPHeader{
		Timestamp:  r.Header.Get("X-TIMESTAMP"),
		Signature:  r.Header.Get("X-SIGNATURE"),
		Origin:     r.Header.Get("ORIGIN"),
		PartnerID:  r.Header.Get("X-PARTNER-ID"),
		ExternalID: r.Header.Get("X-EXTERNAL-ID"),
		IPAddress:  r.Header.Get("X-IP-ADDRESS"),
		DeviceID:   r.Header.Get("X-DEVICE-ID"),
		Latitude:   r.Header.Get("X-LATITUDE"),
		Longitude:  r.Header.Get("X-LONGITUDE"),
		ChannelID:  r.Header.Get("CHANNEL-ID"),
	}
}

var (
	SnapSuccess = SnapDetail{"Success", 200, "00", "Successful", "Successful", false}
	SnapPending = SnapDetail{"Success", 202, "00", "Request In Progress", "Transaction still on process", false}

	SnapBadRequest     = SnapDetail{"System", 400, "00", "Bad Request", "General request failed error", false}
	SnapInvalidFormat  = SnapDetail{"Message", 400, "01", "Invalid Field Format", "Invalid format", true}
	SnapMandatoryField = SnapDetail{"Message", 400, "02", "Invalid Mandatory Field", "Missing mandatory field", true}
	SnapDuplicateExtID = SnapDetail{"System", 409, "00", "Conflict", "Cannot use same X-EXTERNAL-ID in same day", false}
	SnapDuplicateRefNo = SnapDetail{"System", 409, "01", "Duplicate partnerReferenceNo", "Transaction already success", false}

	SnapUnauthorized = SnapDetail{"System", 401, "00", "Unauthorized,", "General unauthorized error", true}
	SnapInvalidToken = SnapDetail{"System", 401, "01", "Invalid Token (B2B)", "Access Token invalid/expired", false}

	SnapExceedLimit  = SnapDetail{"Business", 403, "02", "Exceeds Transaction Amount Limit", "Exceeds limit", false}
	SnapInsufficient = SnapDetail{"Business", 403, "14", "Insufficient Funds", "Insufficient Funds", false}

	SnapTrxNotFound    = SnapDetail{"Business", 404, "01", "Transaction Not Found", "Transaction not found", false}
	SnapInvalidAccount = SnapDetail{"Business", 404, "11", "Invalid Card/Account/Customer", "Account may be blacklisted/invalid", false}
	SnapInvalidAmount  = SnapDetail{"Business", 404, "13", "Invalid Amount", "Amount doesn't match", false}

	SnapInternalError = SnapDetail{"System", 500, "01", "Internal Server Error", "Unknown Internal Server Failure", false}
	SnapExternalError = SnapDetail{"System", 500, "02", "External Server Error", "Backend system failure", false}
	SnapTimeout       = SnapDetail{"System", 504, "00", "Timeout", "Timeout from the issuer", false}
	SnapSuspiciousTransaction = SnapDetail{"Business", 403, "01", "Suspicious Transaction", "Transaction rejected by Fraud Detection System", false}
)

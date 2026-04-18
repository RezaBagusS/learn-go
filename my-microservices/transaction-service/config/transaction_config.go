package config

import "time"

const (
	REDIS_KEY_TRANSACTION_LIST    = "transaction_list"
	REDIS_KEY_TRANSACTION_ID      = "transaction_id"
	REDIS_KEY_TRANSACTION_SUMMARY = "transaction_summary"
	REDIS_KEY_ACCESS_TOKEN        = "access_token"
	REDIS_KEY_FRAUD_FALLBACK      = "fraud_fallback"

	DOMAIN_TRANSACTION = "transaction"
)

const (
	SVC_CODE_TRX_HISTORY_LIST   = "12"
	SVC_CODE_TRX_HISTORY_DETAIL = "13"
	SVC_CODE_BANK_STATEMENT     = "14"
	SVC_CODE_TRANSFER_INTRABANK = "17"
	SVC_CODE_TRANSFER_INTERBANK = "18"
)

const (
	TimeCache = 30 * time.Minute
	TimeLock  = 24 * time.Hour
)

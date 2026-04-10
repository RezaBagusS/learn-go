package config

const (
	REDIS_KEY_BANK_LIST           = "bank_list"
	REDIS_KEY_BANK_ID             = "bank_id"
	REDIS_KEY_ACCOUNT_LIST        = "account_list"
	REDIS_KEY_ACCOUNT_ID          = "account_id"
	REDIS_KEY_ACCOUNT_TRANSACTION = "account_transaction"
	REDIS_KEY_TRANSACTION_LIST    = "transaction_list"
	REDIS_KEY_TRANSACTION_SUMMARY = "transaction_summary"
	REDIS_KEY_TRANSACTION_ID      = "transaction_id"
	REDIS_KEY_ACCESS_TOKEN        = "access_token"

	DOMAIN_BANK        = "bank"
	DOMAIN_ACCOUNT     = "account"
	DOMAIN_TRANSACTION = "transaction"
	DOMAIN_OAUTH       = "oaut"
)

const (
	SVC_CODE_CARD_REGISTRATION        = "01"
	SVC_CODE_CARD_BIND_LIMIT          = "02"
	SVC_CODE_CARD_INQUIRY             = "03"
	SVC_CODE_OTP_VERIFICATION         = "04"
	SVC_CODE_CARD_UNBIND              = "05"
	SVC_CODE_ACCOUNT_CREATION         = "06"
	SVC_CODE_ACCOUNT_BIND             = "07"
	SVC_CODE_ACCOUNT_INQUIRY          = "08"
	SVC_CODE_ACCOUNT_UNBIND           = "09"
	SVC_CODE_AUTH_CODE                = "10"
	SVC_CODE_BALANCE_INQUIRY          = "11"
	SVC_CODE_TRX_HISTORY_LIST         = "12"
	SVC_CODE_TRX_HISTORY_DETAIL       = "13"
	SVC_CODE_BANK_STATEMENT           = "14"
	SVC_CODE_ACCOUNT_INQUIRY_INTERNAL = "15"
	SVC_CODE_ACCOUNT_INQUIRY_EXTERNAL = "16"
	SVC_CODE_TRANSFER_INTRABANK       = "17"
	SVC_CODE_TRANSFER_INTERBANK       = "18"
	SVC_CODE_TRANSFER_REQUEST_PAYMENT = "19"
)

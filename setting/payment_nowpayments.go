package setting

import "github.com/QuantumNous/new-api/common"

var (
	NowPaymentsEnabled     = common.GetEnvOrDefaultBool("NOWPAYMENTS_ENABLED", false)
	NowPaymentsApiKey      = common.GetEnvOrDefaultString("NOWPAYMENTS_API_KEY", "")
	NowPaymentsIPNSecret   = common.GetEnvOrDefaultString("NOWPAYMENTS_IPN_SECRET", "")
	NowPaymentsBaseURL     = common.GetEnvOrDefaultString("NOWPAYMENTS_BASE_URL", "https://api.nowpayments.io/v1")
	NowPaymentsCallbackURL = common.GetEnvOrDefaultString("NOWPAYMENTS_CALLBACK_URL", "")
	NowPaymentsCurrency    = common.GetEnvOrDefaultString("NOWPAYMENTS_CURRENCY", "usdtbsc")
	NowPaymentsCurrencies  = common.GetEnvOrDefaultString("NOWPAYMENTS_CURRENCIES", NowPaymentsCurrency)
	NowPaymentsMinTopUp    = common.GetEnvOrDefault("NOWPAYMENTS_MIN_TOPUP", 1)
)

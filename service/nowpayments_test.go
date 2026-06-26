package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withNowPaymentsSettingsForTest(t *testing.T) {
	t.Helper()
	originalCurrency := setting.NowPaymentsCurrency
	originalCurrencies := setting.NowPaymentsCurrencies
	originalBaseURL := setting.NowPaymentsBaseURL
	originalAPIKey := setting.NowPaymentsApiKey
	t.Cleanup(func() {
		setting.NowPaymentsCurrency = originalCurrency
		setting.NowPaymentsCurrencies = originalCurrencies
		setting.NowPaymentsBaseURL = originalBaseURL
		setting.NowPaymentsApiKey = originalAPIKey
	})
}

func TestGetNowPaymentsCurrencies_NormalizesDedupesAndPrependsFallback(t *testing.T) {
	withNowPaymentsSettingsForTest(t)

	setting.NowPaymentsCurrency = "USDTBSC"
	setting.NowPaymentsCurrencies = " eth, USDTBSC, bnbbsc, eth, , sol "

	assert.Equal(t, []string{"eth", "usdtbsc", "bnbbsc", "sol"}, GetNowPaymentsCurrencies())
	assert.Equal(t, "eth", NormalizeNowPaymentsCurrency(" ETH "))
	assert.True(t, IsAllowedNowPaymentsCurrency("BNBBSC"))
	assert.False(t, IsAllowedNowPaymentsCurrency("usdterc20"))
}

func TestGetNowPaymentsCurrencies_FallsBackWhenListIsEmpty(t *testing.T) {
	withNowPaymentsSettingsForTest(t)

	setting.NowPaymentsCurrency = "ETH"
	setting.NowPaymentsCurrencies = " , "

	assert.Equal(t, []string{"eth"}, GetNowPaymentsCurrencies())
	assert.Equal(t, "eth", NormalizeNowPaymentsCurrency(""))
	assert.True(t, IsAllowedNowPaymentsCurrency(""))
}

func TestEstimateNowPaymentsPrice_UsesSelectedCurrencyAndApiKey(t *testing.T) {
	withNowPaymentsSettingsForTest(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/estimate", r.URL.Path)
		require.Equal(t, "secret-key", r.Header.Get("x-api-key"))
		require.Equal(t, "7.30", r.URL.Query().Get("amount"))
		require.Equal(t, "usd", r.URL.Query().Get("currency_from"))
		require.Equal(t, "eth", r.URL.Query().Get("currency_to"))
		_, _ = w.Write([]byte(`{"estimated_amount":0.0021}`))
	}))
	t.Cleanup(server.Close)

	setting.NowPaymentsApiKey = "secret-key"
	setting.NowPaymentsBaseURL = server.URL
	setting.NowPaymentsCurrency = "usdtbsc"

	estimate, err := EstimateNowPaymentsPrice(context.Background(), 7.3, "ETH")
	require.NoError(t, err)
	assert.Equal(t, "usd", estimate.CurrencyFrom)
	assert.Equal(t, "eth", estimate.CurrencyTo)
	assert.Equal(t, 7.3, estimate.AmountFrom)
	assert.Equal(t, 0.0021, estimate.EstimatedAmount)
}

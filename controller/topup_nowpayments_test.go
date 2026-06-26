package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestNowPaymentsAmount_RequiresEnabledTopUpBeforeParsing(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	originalEnabled := setting.NowPaymentsEnabled
	originalAPIKey := setting.NowPaymentsApiKey
	originalIPNSecret := setting.NowPaymentsIPNSecret
	t.Cleanup(func() {
		setting.NowPaymentsEnabled = originalEnabled
		setting.NowPaymentsApiKey = originalAPIKey
		setting.NowPaymentsIPNSecret = originalIPNSecret
	})

	setting.NowPaymentsEnabled = false
	setting.NowPaymentsApiKey = "api_key"
	setting.NowPaymentsIPNSecret = "ipn_secret"

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/user/nowpayments/amount", bytes.NewBufferString(`{"amount":1,"pay_currency":"eth"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 1)

	RequestNowPaymentsAmount(ctx)

	var response struct {
		Message string `json:"message"`
		Data    string `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "error", response.Message)
	assert.Equal(t, "NOWPayments 配置不完整", response.Data)
}

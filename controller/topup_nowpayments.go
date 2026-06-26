package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

type NowPaymentsPayRequest struct {
	Amount      int64  `json:"amount"`
	PayCurrency string `json:"pay_currency"`
}

func resolveNowPaymentsPayCurrency(c *gin.Context, rawCurrency string) (string, bool) {
	payCurrency := service.NormalizeNowPaymentsCurrency(rawCurrency)
	if !service.IsAllowedNowPaymentsCurrency(payCurrency) {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "不支持的 NOWPayments 支付币种"})
		return "", false
	}
	return payCurrency, true
}

func getNowPaymentsMinTopUp() int64 {
	minTopUp := setting.NowPaymentsMinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		minTopUp = int(decimal.NewFromInt(int64(minTopUp)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
	}
	return int64(minTopUp)
}

func normalizeNowPaymentsTopUpAmount(amount int64) int64 {
	if operation_setting.GetQuotaDisplayType() != operation_setting.QuotaDisplayTypeTokens {
		return amount
	}
	normalized := decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	if normalized < 1 {
		return 1
	}
	return normalized
}

func RequestNowPaymentsAmount(c *gin.Context) {
	if !isNowPaymentsTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "NOWPayments 配置不完整"})
		return
	}

	var req NowPaymentsPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getNowPaymentsMinTopUp() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getNowPaymentsMinTopUp())})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getNowPaymentsPayMoney(req.Amount, group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	payCurrency, ok := resolveNowPaymentsPayCurrency(c, req.PayCurrency)
	if !ok {
		return
	}
	estimate, err := service.EstimateNowPaymentsPrice(c.Request.Context(), payMoney, payCurrency)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("NOWPayments 获取实时估价失败 user_id=%d amount=%d money=%.2f pay_currency=%s error=%q", id, req.Amount, payMoney, payCurrency, err.Error()))
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"data": gin.H{
				"price_amount":   strconv.FormatFloat(payMoney, 'f', 2, 64),
				"price_currency": "usd",
				"pay_currency":   payCurrency,
				"estimated":      false,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"price_amount":     strconv.FormatFloat(payMoney, 'f', 2, 64),
			"price_currency":   strings.ToLower(strings.TrimSpace(estimate.CurrencyFrom)),
			"pay_amount":       strconv.FormatFloat(estimate.EstimatedAmount, 'f', -1, 64),
			"pay_currency":     strings.ToLower(strings.TrimSpace(estimate.CurrencyTo)),
			"estimated":        true,
			"estimate_message": "NOWPayments live estimate. Final crypto amount is confirmed on the hosted checkout page.",
		},
	})
}

func RequestNowPaymentsPay(c *gin.Context) {
	if !isNowPaymentsTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "NOWPayments 配置不完整"})
		return
	}

	var req NowPaymentsPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getNowPaymentsMinTopUp() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getNowPaymentsMinTopUp())})
		return
	}

	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "用户不存在"})
		return
	}
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}

	payMoney := getNowPaymentsPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	payCurrency, ok := resolveNowPaymentsPayCurrency(c, req.PayCurrency)
	if !ok {
		return
	}

	tradeNo := fmt.Sprintf("NOWPAYMENTS-%d-%d-%s", id, time.Now().UnixMilli(), randstr.String(6))
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          normalizeNowPaymentsTopUpAmount(req.Amount),
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodNowPayments,
		PaymentProvider: model.PaymentProviderNowPayments,
		PayCurrency:     payCurrency,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("NOWPayments 创建充值订单失败 user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	invoice, err := service.CreateNowPaymentsInvoice(c.Request.Context(), &service.NowPaymentsCreateInvoiceParams{
		PriceAmount:      payMoney,
		PayCurrency:      payCurrency,
		CallbackURL:      service.ResolveNowPaymentsCallbackURL(),
		OrderID:          tradeNo,
		OrderDescription: fmt.Sprintf("new-api prepaid top-up %d", req.Amount),
		SuccessURL:       paymentReturnPath("/console/topup"),
		CancelURL:        paymentReturnPath("/console/topup"),
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("NOWPayments 创建托管支付失败 user_id=%d trade_no=%s amount=%d error=%q", id, tradeNo, req.Amount, err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("NOWPayments 充值订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f pay_currency=%s payment_url=%q", id, tradeNo, req.Amount, payMoney, payCurrency, invoice.PaymentURL))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"payment_url": invoice.PaymentURL,
			"order_id":    tradeNo,
		},
	})
}

func NowPaymentsWebhook(c *gin.Context) {
	if !isNowPaymentsWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("NOWPayments webhook 被拒绝 reason=webhook_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("NOWPayments webhook 读取请求体失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	signature := c.GetHeader("x-nowpayments-sig")
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("NOWPayments webhook 收到请求 path=%q client_ip=%s body_size=%d has_signature=%t", c.Request.RequestURI, c.ClientIP(), len(bodyBytes), signature != ""))

	if err := service.VerifyNowPaymentsSignature(bodyBytes, signature); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("NOWPayments webhook 验签失败 path=%q client_ip=%s error=%q", c.Request.RequestURI, c.ClientIP(), err.Error()))
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	event, err := service.ParseNowPaymentsWebhook(bodyBytes)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("NOWPayments webhook 解析失败 path=%q client_ip=%s error=%q body_size=%d", c.Request.RequestURI, c.ClientIP(), err.Error(), len(bodyBytes)))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	tradeNo := strings.TrimSpace(event.OrderID)
	status := event.NormalizedStatus()
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("NOWPayments webhook 验签成功 trade_no=%s payment_status=%s client_ip=%s", tradeNo, status, c.ClientIP()))
	if tradeNo == "" {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("NOWPayments webhook 缺少订单号 payment_status=%s client_ip=%s", status, c.ClientIP()))
		c.Status(http.StatusOK)
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	if event.IsSuccessStatus() {
		err = model.CompleteNowPaymentsTopUp(tradeNo, c.ClientIP(), event.PriceAmount, event.PriceCurrency, event.PayCurrency, "")
		switch {
		case err == nil:
			logger.LogInfo(c.Request.Context(), fmt.Sprintf("NOWPayments 充值成功 trade_no=%s payment_status=%s client_ip=%s", tradeNo, status, c.ClientIP()))
			c.Status(http.StatusOK)
			return
		case errors.Is(err, model.ErrTopUpNotFound), errors.Is(err, model.ErrPaymentMethodMismatch), errors.Is(err, model.ErrTopUpStatusInvalid):
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("NOWPayments webhook 无需重试 trade_no=%s payment_status=%s client_ip=%s error=%q", tradeNo, status, c.ClientIP(), err.Error()))
			c.Status(http.StatusOK)
			return
		default:
			logger.LogError(c.Request.Context(), fmt.Sprintf("NOWPayments 充值处理失败 trade_no=%s payment_status=%s client_ip=%s error=%q", tradeNo, status, c.ClientIP(), err.Error()))
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}

	if failedStatus := event.FailureStatus(); failedStatus != "" {
		err = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderNowPayments, failedStatus)
		switch {
		case err == nil:
			logger.LogInfo(c.Request.Context(), fmt.Sprintf("NOWPayments 充值订单状态更新成功 trade_no=%s payment_status=%s target_status=%s client_ip=%s", tradeNo, status, failedStatus, c.ClientIP()))
		case errors.Is(err, model.ErrTopUpNotFound), errors.Is(err, model.ErrPaymentMethodMismatch), errors.Is(err, model.ErrTopUpStatusInvalid):
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("NOWPayments 失败订单状态无需重试 trade_no=%s payment_status=%s target_status=%s client_ip=%s error=%q", tradeNo, status, failedStatus, c.ClientIP(), err.Error()))
		default:
			logger.LogError(c.Request.Context(), fmt.Sprintf("NOWPayments 更新失败订单状态失败 trade_no=%s payment_status=%s target_status=%s client_ip=%s error=%q", tradeNo, status, failedStatus, c.ClientIP(), err.Error()))
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("NOWPayments webhook 忽略非终态事件 trade_no=%s payment_status=%s client_ip=%s", tradeNo, status, c.ClientIP()))
	c.Status(http.StatusOK)
}

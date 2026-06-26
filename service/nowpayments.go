package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
)

type NowPaymentsCreateInvoiceParams struct {
	PriceAmount      float64
	PayCurrency      string
	CallbackURL      string
	OrderID          string
	OrderDescription string
	SuccessURL       string
	CancelURL        string
}

type NowPaymentsInvoice struct {
	ID         string
	OrderID    string
	PaymentURL string
}

type NowPaymentsEstimate struct {
	CurrencyFrom    string  `json:"currency_from"`
	AmountFrom      float64 `json:"amount_from"`
	CurrencyTo      string  `json:"currency_to"`
	EstimatedAmount float64 `json:"estimated_amount"`
}

type NowPaymentsWebhookEvent struct {
	PaymentID        int64   `json:"payment_id"`
	PurchaseID       string  `json:"purchase_id"`
	InvoiceID        string  `json:"invoice_id"`
	OrderID          string  `json:"order_id"`
	OrderDescription string  `json:"order_description"`
	PaymentStatus    string  `json:"payment_status"`
	PriceAmount      float64 `json:"price_amount"`
	PriceCurrency    string  `json:"price_currency"`
	PayAmount        float64 `json:"pay_amount"`
	PayCurrency      string  `json:"pay_currency"`
	ActuallyPaid     float64 `json:"actually_paid"`
	OutcomeAmount    float64 `json:"outcome_amount"`
	OutcomeCurrency  string  `json:"outcome_currency"`
}

func (e *NowPaymentsWebhookEvent) NormalizedStatus() string {
	if e == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(e.PaymentStatus))
}

func (e *NowPaymentsWebhookEvent) IsSuccessStatus() bool {
	switch e.NormalizedStatus() {
	case "finished", "paid", "completed":
		return true
	default:
		return false
	}
}

func (e *NowPaymentsWebhookEvent) FailureStatus() string {
	switch e.NormalizedStatus() {
	case "failed":
		return common.TopUpStatusFailed
	case "expired":
		return common.TopUpStatusExpired
	default:
		return ""
	}
}

func ResolveNowPaymentsCallbackURL() string {
	callbackAddress := strings.TrimRight(GetCallbackAddress(), "/")
	if callbackAddress != "" {
		return callbackAddress + "/api/nowpayments/callback"
	}
	return strings.TrimSpace(setting.NowPaymentsCallbackURL)
}

func GetNowPaymentsCurrency() string {
	currency := strings.ToLower(strings.TrimSpace(setting.NowPaymentsCurrency))
	if currency == "" {
		return "usdtbsc"
	}
	return currency
}

func GetNowPaymentsCurrencies() []string {
	seen := make(map[string]bool)
	currencies := make([]string, 0)
	for _, rawCurrency := range strings.Split(setting.NowPaymentsCurrencies, ",") {
		currency := strings.ToLower(strings.TrimSpace(rawCurrency))
		if currency == "" || seen[currency] {
			continue
		}
		seen[currency] = true
		currencies = append(currencies, currency)
	}

	fallback := GetNowPaymentsCurrency()
	if len(currencies) == 0 {
		return []string{fallback}
	}
	if !seen[fallback] {
		currencies = append([]string{fallback}, currencies...)
	}
	return currencies
}

func NormalizeNowPaymentsCurrency(payCurrency string) string {
	currency := strings.ToLower(strings.TrimSpace(payCurrency))
	if currency == "" {
		return GetNowPaymentsCurrency()
	}
	return currency
}

func IsAllowedNowPaymentsCurrency(payCurrency string) bool {
	currency := NormalizeNowPaymentsCurrency(payCurrency)
	for _, allowed := range GetNowPaymentsCurrencies() {
		if currency == allowed {
			return true
		}
	}
	return false
}

func CreateNowPaymentsInvoice(ctx context.Context, params *NowPaymentsCreateInvoiceParams) (*NowPaymentsInvoice, error) {
	if params == nil {
		return nil, fmt.Errorf("missing invoice params")
	}
	if strings.TrimSpace(setting.NowPaymentsApiKey) == "" {
		return nil, fmt.Errorf("missing NOWPayments API key")
	}
	payCurrency := strings.ToLower(strings.TrimSpace(params.PayCurrency))
	if payCurrency == "" {
		payCurrency = GetNowPaymentsCurrency()
	}
	body, err := common.Marshal(map[string]any{
		"price_amount":      params.PriceAmount,
		"price_currency":    "usd",
		"pay_currency":      payCurrency,
		"ipn_callback_url":  strings.TrimSpace(params.CallbackURL),
		"order_id":          strings.TrimSpace(params.OrderID),
		"order_description": strings.TrimSpace(params.OrderDescription),
		"success_url":       strings.TrimSpace(params.SuccessURL),
		"cancel_url":        strings.TrimSpace(params.CancelURL),
		"is_fixed_rate":     true,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal NOWPayments invoice request: %w", err)
	}

	endpoint := strings.TrimRight(strings.TrimSpace(setting.NowPaymentsBaseURL), "/") + "/invoice"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create NOWPayments invoice request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", setting.NowPaymentsApiKey)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("request NOWPayments invoice: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read NOWPayments invoice response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("NOWPayments invoice request failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var payload struct {
		ID         any    `json:"id"`
		OrderID    string `json:"order_id"`
		InvoiceURL string `json:"invoice_url"`
		PaymentURL string `json:"payment_url"`
		URL        string `json:"url"`
	}
	if err := common.Unmarshal(responseBody, &payload); err != nil {
		return nil, fmt.Errorf("decode NOWPayments invoice response: %w", err)
	}

	paymentURL := strings.TrimSpace(payload.InvoiceURL)
	if paymentURL == "" {
		paymentURL = strings.TrimSpace(payload.PaymentURL)
	}
	if paymentURL == "" {
		paymentURL = strings.TrimSpace(payload.URL)
	}
	if paymentURL == "" {
		return nil, fmt.Errorf("NOWPayments invoice response missing payment url")
	}
	parsedURL, err := url.Parse(paymentURL)
	if err != nil || !parsedURL.IsAbs() {
		return nil, fmt.Errorf("NOWPayments invoice returned invalid payment url")
	}

	return &NowPaymentsInvoice{
		ID:         fmt.Sprint(payload.ID),
		OrderID:    strings.TrimSpace(payload.OrderID),
		PaymentURL: paymentURL,
	}, nil
}

func EstimateNowPaymentsPrice(ctx context.Context, amount float64, payCurrency string) (*NowPaymentsEstimate, error) {
	if strings.TrimSpace(setting.NowPaymentsApiKey) == "" {
		return nil, fmt.Errorf("missing NOWPayments API key")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("invalid estimate amount")
	}
	currencyTo := strings.ToLower(strings.TrimSpace(payCurrency))
	if currencyTo == "" {
		currencyTo = GetNowPaymentsCurrency()
	}

	endpoint, err := url.Parse(strings.TrimRight(strings.TrimSpace(setting.NowPaymentsBaseURL), "/") + "/estimate")
	if err != nil {
		return nil, fmt.Errorf("create NOWPayments estimate url: %w", err)
	}
	query := endpoint.Query()
	query.Set("amount", strconv.FormatFloat(amount, 'f', 2, 64))
	query.Set("currency_from", "usd")
	query.Set("currency_to", currencyTo)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create NOWPayments estimate request: %w", err)
	}
	req.Header.Set("x-api-key", setting.NowPaymentsApiKey)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("request NOWPayments estimate: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read NOWPayments estimate response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("NOWPayments estimate request failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var estimate NowPaymentsEstimate
	if err := common.Unmarshal(responseBody, &estimate); err != nil {
		return nil, fmt.Errorf("decode NOWPayments estimate response: %w", err)
	}
	if estimate.EstimatedAmount <= 0 {
		return nil, fmt.Errorf("NOWPayments estimate response missing estimated amount")
	}
	if strings.TrimSpace(estimate.CurrencyTo) == "" {
		estimate.CurrencyTo = currencyTo
	}
	if strings.TrimSpace(estimate.CurrencyFrom) == "" {
		estimate.CurrencyFrom = "usd"
	}
	if estimate.AmountFrom == 0 {
		estimate.AmountFrom = amount
	}
	return &estimate, nil
}

func ParseNowPaymentsWebhook(payload []byte) (*NowPaymentsWebhookEvent, error) {
	var event NowPaymentsWebhookEvent
	if err := common.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func VerifyNowPaymentsSignature(payload []byte, signature string) error {
	secret := strings.TrimSpace(setting.NowPaymentsIPNSecret)
	if secret == "" {
		return fmt.Errorf("missing NOWPayments IPN secret")
	}
	canonicalBody, err := canonicalizeNowPaymentsJSON(json.RawMessage(bytes.TrimSpace(payload)))
	if err != nil {
		return fmt.Errorf("canonicalize NOWPayments webhook body: %w", err)
	}
	mac := hmac.New(sha512.New, []byte(secret))
	_, _ = mac.Write(canonicalBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	actual := strings.ToLower(strings.TrimSpace(signature))
	if !hmac.Equal([]byte(actual), []byte(expected)) {
		return fmt.Errorf("invalid NOWPayments signature")
	}
	return nil
}

func canonicalizeNowPaymentsJSON(raw json.RawMessage) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	switch common.GetJsonType(trimmed) {
	case "object":
		var object map[string]json.RawMessage
		if err := common.Unmarshal(trimmed, &object); err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		var buf bytes.Buffer
		buf.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodedKey, err := common.Marshal(key)
			if err != nil {
				return nil, err
			}
			buf.Write(encodedKey)
			buf.WriteByte(':')
			encodedValue, err := canonicalizeNowPaymentsJSON(object[key])
			if err != nil {
				return nil, err
			}
			buf.Write(encodedValue)
		}
		buf.WriteByte('}')
		return buf.Bytes(), nil
	case "array":
		var array []json.RawMessage
		if err := common.Unmarshal(trimmed, &array); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, item := range array {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodedItem, err := canonicalizeNowPaymentsJSON(item)
			if err != nil {
				return nil, err
			}
			buf.Write(encodedItem)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil
	case "string":
		var value string
		if err := common.Unmarshal(trimmed, &value); err != nil {
			return nil, err
		}
		return common.Marshal(value)
	case "number", "boolean", "null":
		return trimmed, nil
	default:
		return nil, fmt.Errorf("unsupported JSON payload")
	}
}

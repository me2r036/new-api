/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useState, useCallback } from 'react'
import i18next from 'i18next'
import { toast } from 'sonner'
import {
  calculateAmount,
  calculateNowPaymentsAmount,
  calculateStripeAmount,
  calculateWaffoPancakeAmount,
  requestPayment,
  requestStripePayment,
  isApiSuccess,
} from '../api'
import type { NowPaymentsAmountEstimate } from '../types'
import {
  isNowPaymentsPayment,
  isStripePayment,
  isWaffoPancakePayment,
  submitPaymentForm,
} from '../lib'

// ============================================================================
// Payment Hook
// ============================================================================

export function usePayment() {
  const [amount, setAmount] = useState<number>(0)
  const [nowPaymentsEstimate, setNowPaymentsEstimate] =
    useState<NowPaymentsAmountEstimate | null>(null)
  const [calculating, setCalculating] = useState(false)
  const [processing, setProcessing] = useState(false)

  // Calculate payment amount
  const calculatePaymentAmount = useCallback(
    async (topupAmount: number, paymentType: string, payCurrency?: string) => {
      try {
        setCalculating(true)

        const isStripe = isStripePayment(paymentType)
        const isNowPayments = isNowPaymentsPayment(paymentType)
        const isPancake = isWaffoPancakePayment(paymentType)
        const resolvedNowPaymentsCurrency =
          payCurrency ||
          (paymentType.startsWith('nowpayments:')
            ? paymentType.split(':')[1]
            : undefined)
        if (!isNowPayments) {
          setNowPaymentsEstimate(null)
        }
        const response = isStripe
          ? await calculateStripeAmount({ amount: topupAmount })
          : isNowPayments
            ? await calculateNowPaymentsAmount({
                amount: topupAmount,
                pay_currency: resolvedNowPaymentsCurrency,
              })
          : isPancake
            ? await calculateWaffoPancakeAmount({ amount: topupAmount })
            : await calculateAmount({ amount: topupAmount })

        if (isApiSuccess(response) && response.data) {
          if (isNowPayments && typeof response.data === 'object') {
            setNowPaymentsEstimate(response.data)
            const calculatedAmount = parseFloat(
              response.data.pay_amount || response.data.price_amount || '0'
            )
            setAmount(calculatedAmount)
            return calculatedAmount
          }

          setNowPaymentsEstimate(null)
          const calculatedAmount = parseFloat(String(response.data))
          setAmount(calculatedAmount)
          return calculatedAmount
        }

        // Don't show error for calculation, just set to 0
        if (isNowPayments) {
          setNowPaymentsEstimate(null)
        }
        setAmount(0)
        return 0
      } catch (_error) {
        setNowPaymentsEstimate(null)
        setAmount(0)
        return 0
      } finally {
        setCalculating(false)
      }
    },
    []
  )

  // Process payment
  const processPayment = useCallback(
    async (topupAmount: number, paymentType: string) => {
      try {
        setProcessing(true)

        const isStripe = isStripePayment(paymentType)
        const amount = Math.floor(topupAmount)

        const response = isStripe
          ? await requestStripePayment({
              amount,
              payment_method: 'stripe',
            })
          : await requestPayment({
              amount,
              payment_method: paymentType,
            })

        if (!isApiSuccess(response)) {
          toast.error(response.message || i18next.t('Payment request failed'))
          return false
        }

        // Handle Stripe payment
        if (isStripe && response.data?.pay_link) {
          window.open(response.data.pay_link as string, '_blank')
          toast.success(i18next.t('Redirecting to payment page...'))
          return true
        }

        // Handle non-Stripe payment
        if (!isStripe && response.data) {
          const url = (response as unknown as { url?: string }).url
          if (url) {
            submitPaymentForm(url, response.data)
            toast.success(i18next.t('Redirecting to payment page...'))
            return true
          }
        }

        return false
      } catch (_error) {
        toast.error(i18next.t('Payment request failed'))
        return false
      } finally {
        setProcessing(false)
      }
    },
    []
  )

  return {
    amount,
    nowPaymentsEstimate,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
    setAmount,
  }
}

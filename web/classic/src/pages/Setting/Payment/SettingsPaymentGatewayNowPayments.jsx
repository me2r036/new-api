/*
Copyright (C) 2025 QuantumNous

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

import React, { useEffect, useRef, useState } from 'react';
import { Banner, Button, Col, Form, Row, Spin } from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
  toBoolean,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { BookOpen } from 'lucide-react';

export default function SettingsPaymentGatewayNowPayments(props) {
  const { t } = useTranslation();
  const sectionTitle = props.hideSectionTitle
    ? undefined
    : t('NOWPayments 设置');
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    NowPaymentsEnabled: false,
    NowPaymentsApiKey: '',
    NowPaymentsIPNSecret: '',
    NowPaymentsCurrency: 'usdtbsc',
    NowPaymentsCurrencies: 'usdtbsc,eth,bnbbsc',
    NowPaymentsMinTopUp: 1,
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        NowPaymentsEnabled: toBoolean(props.options.NowPaymentsEnabled),
        NowPaymentsApiKey: props.options.NowPaymentsApiKey || '',
        NowPaymentsIPNSecret: props.options.NowPaymentsIPNSecret || '',
        NowPaymentsCurrency: String(
          props.options.NowPaymentsCurrency || 'usdtbsc',
        ).toLowerCase(),
        NowPaymentsCurrencies: String(
          props.options.NowPaymentsCurrencies || 'usdtbsc,eth,bnbbsc',
        ).toLowerCase(),
        NowPaymentsMinTopUp:
          parseFloat(props.options.NowPaymentsMinTopUp) || 1,
      };
      setInputs(currentInputs);
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs({
      ...values,
      NowPaymentsCurrency: String(
        values.NowPaymentsCurrency || 'usdtbsc',
      ).toLowerCase(),
      NowPaymentsCurrencies: String(
        values.NowPaymentsCurrencies || 'usdtbsc',
      ).toLowerCase(),
    });
  };

  const submitNowPaymentsSetting = async () => {
    setLoading(true);
    try {
      const options = [
        {
          key: 'NowPaymentsEnabled',
          value: inputs.NowPaymentsEnabled ? 'true' : 'false',
        },
        {
          key: 'NowPaymentsCurrency',
          value: String(inputs.NowPaymentsCurrency || 'usdtbsc').toLowerCase(),
        },
        {
          key: 'NowPaymentsCurrencies',
          value: String(inputs.NowPaymentsCurrencies || 'usdtbsc')
            .split(',')
            .map((currency) => currency.trim().toLowerCase())
            .filter(Boolean)
            .join(','),
        },
        {
          key: 'NowPaymentsMinTopUp',
          value: String(inputs.NowPaymentsMinTopUp || 1),
        },
      ];

      if (inputs.NowPaymentsApiKey) {
        options.push({
          key: 'NowPaymentsApiKey',
          value: inputs.NowPaymentsApiKey,
        });
      }

      if (inputs.NowPaymentsIPNSecret) {
        options.push({
          key: 'NowPaymentsIPNSecret',
          value: inputs.NowPaymentsIPNSecret,
        });
      }

      const results = await Promise.all(
        options.map((opt) =>
          API.put('/api/option/', {
            key: opt.key,
            value: opt.value,
          }),
        ),
      );

      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => showError(res.data.message));
      } else {
        showSuccess(t('更新成功'));
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={sectionTitle}>
          <Banner
            type='info'
            icon={<BookOpen size={16} />}
            description={
              <>
                {t('NOWPayments 使用托管加密货币结账页。')}
                <br />
                {t('回调地址')}：
                {props.options.ServerAddress
                  ? removeTrailingSlash(props.options.ServerAddress)
                  : t('网站地址')}
                /api/nowpayments/callback
              </>
            }
            style={{ marginBottom: 16 }}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='NowPaymentsEnabled'
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                label={t('启用 NOWPayments 充值')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='NowPaymentsApiKey'
                label={t('API 密钥')}
                placeholder={t('留空表示保持当前不变')}
                extraText={t('保存后不会回显，请填写 NOWPayments API Key')}
                type='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='NowPaymentsIPNSecret'
                label={t('IPN 密钥')}
                placeholder={t('留空表示保持当前不变')}
                extraText={t('用于校验 NOWPayments 回调签名，保存后不会回显')}
                type='password'
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='NowPaymentsCurrency'
                label={t('支付币种')}
                placeholder='usdtbsc'
                extraText={t('默认 NOWPayments 加密货币代码，例如 usdtbsc')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='NowPaymentsCurrencies'
                label={t('允许支付币种')}
                placeholder='usdtbsc,eth,bnbbsc'
                extraText={t('逗号分隔的 NOWPayments 加密货币代码，例如 usdtbsc,eth,bnbbsc')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='NowPaymentsMinTopUp'
                precision={2}
                label={t('最低充值美元数量')}
                placeholder={t('例如：1，就是最低充值1$')}
                extraText={t('用户单次最少可充值的美元数量')}
              />
            </Col>
          </Row>

          <Button onClick={submitNowPaymentsSetting} style={{ marginTop: 16 }}>
            {t('更新 NOWPayments 设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}

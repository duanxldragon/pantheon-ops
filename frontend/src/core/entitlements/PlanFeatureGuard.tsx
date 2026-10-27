import React from 'react';
import { Button, Result, Space, Tag } from '@arco-design/web-react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { getPlanFeature, type PlanFeatureKey } from './plans';
import { usePlanEntitlements } from './usePlanEntitlements';

interface PlanFeatureBoundaryProps {
  feature: PlanFeatureKey;
  children: React.ReactElement;
}

function PlanFeatureBoundary({ feature, children }: Readonly<PlanFeatureBoundaryProps>) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { hasFeature, plan } = usePlanEntitlements();

  if (hasFeature(feature)) {
    return children;
  }

  const definition = getPlanFeature(feature);
  const featureLabel = definition ? t(definition.titleKey) : feature;
  const requiredPlan = definition ? t(`system.tenant.plan.${definition.minPlan}`) : '-';
  const currentPlan = plan ? t(`system.tenant.plan.${plan}`) : '-';

  return (
    <Result
      className="page-result"
      status="warning"
      title={t('system.tenant.planFeature.upgradeTitle')}
      subTitle={t('system.tenant.planFeature.upgradeDescription', {
        feature: featureLabel,
        plan: requiredPlan,
      })}
      extra={
        <Space size={12}>
          <Tag color="orange">
            {t('system.tenant.planFeature.currentPlan', { plan: currentPlan })}
          </Tag>
          <Button type="primary" onClick={() => navigate('/')}>
            {t('common.backHome')}
          </Button>
        </Space>
      }
    />
  );
}

interface PlanFeatureGuardProps {
  feature?: PlanFeatureKey;
  children: React.ReactElement;
}

/**
 * Replaces a plan-gated page with an explicit upgrade state instead of letting
 * a tenant reach a page whose menu entry its plan already hides.
 */
export default function PlanFeatureGuard({ feature, children }: Readonly<PlanFeatureGuardProps>) {
  if (!feature) {
    return children;
  }
  return <PlanFeatureBoundary feature={feature}>{children}</PlanFeatureBoundary>;
}

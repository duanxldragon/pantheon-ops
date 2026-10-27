import { Tag, Typography } from '@arco-design/web-react';
import { IconCheck, IconClose } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import {
  PLAN_FEATURES,
  TENANT_PLANS,
  isPlanAtLeast,
  type TenantPlan,
} from '../../../../core/entitlements/plans';
import './TenantPlanMatrix.css';

interface TenantPlanMatrixProps {
  plan: TenantPlan;
}

/**
 * Renders the plan/feature matrix straight from the frontend entitlement
 * registry. The tenant form never hardcodes feature names or tiers, so adding a
 * paid feature is one registry entry plus the surface declaration on its module.
 */
export default function TenantPlanMatrix({ plan }: Readonly<TenantPlanMatrixProps>) {
  const { t } = useTranslation();

  return (
    <section className="tenant-plan-matrix">
      <header className="tenant-plan-matrix__header">
        <Typography.Text className="tenant-plan-matrix__title">
          {t('system.tenant.planFeature.title')}
        </Typography.Text>
        <Typography.Text type="secondary" className="tenant-plan-matrix__description">
          {t('system.tenant.planFeature.description')}
        </Typography.Text>
      </header>
      <div className="tenant-plan-matrix__scroll">
        <table
          className="tenant-plan-matrix__table"
          aria-label={t('system.tenant.planFeature.title')}
        >
          <thead>
            <tr>
              <th scope="col">{t('system.tenant.planFeature.feature')}</th>
              {TENANT_PLANS.map((item) => (
                <th
                  key={item}
                  scope="col"
                  className={item === plan ? 'tenant-plan-matrix__column--active' : undefined}
                >
                  <span className="tenant-plan-matrix__plan">
                    {t(`system.tenant.plan.${item}`)}
                  </span>
                  {item === plan ? (
                    <Tag size="small" color="arcoblue">
                      {t('system.tenant.planFeature.current')}
                    </Tag>
                  ) : null}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {PLAN_FEATURES.map((feature) => (
              <tr key={feature.key}>
                <th scope="row">
                  <span className="tenant-plan-matrix__feature">{t(feature.titleKey)}</span>
                  <span className="tenant-plan-matrix__feature-description">
                    {t(feature.descriptionKey)}
                  </span>
                </th>
                {TENANT_PLANS.map((item) => {
                  const included = isPlanAtLeast(item, feature.minPlan);
                  return (
                    <td
                      key={item}
                      className={item === plan ? 'tenant-plan-matrix__column--active' : undefined}
                    >
                      <span
                        role="img"
                        aria-label={
                          included
                            ? t('system.tenant.planFeature.included')
                            : t('system.tenant.planFeature.notIncluded')
                        }
                        className={
                          included ? 'tenant-plan-matrix__included' : 'tenant-plan-matrix__excluded'
                        }
                      >
                        {included ? <IconCheck /> : <IconClose />}
                      </span>
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

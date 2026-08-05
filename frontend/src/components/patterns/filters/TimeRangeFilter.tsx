import React, { useCallback, useMemo, useState } from 'react';
import { Button, DatePicker, Typography } from '@arco-design/web-react';
import { IconCalendar } from '@arco-design/web-react/icon';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import './time-range-filter.css';

const RangePicker = DatePicker.RangePicker;

export const TIME_RANGE_FILTER_FORMAT = 'YYYY-MM-DD HH:mm';
// Query payload keeps seconds so `endOf('minute')` really includes the whole
// minute — the display format above would truncate :59 back to :00 and the
// backend's inclusive <= compare would drop the freshest events.
const TIME_RANGE_PAYLOAD_FORMAT = 'YYYY-MM-DD HH:mm:ss';

export interface TimeRangeFilterValue {
  startedAt?: string;
  endedAt?: string;
}

interface TimeRangeFilterProps {
  value: TimeRangeFilterValue;
  onChange: (value: Required<TimeRangeFilterValue>) => void;
}

type TimePreset = {
  labelKey: string;
  minutes?: number;
  hours?: number;
  preset?: 'today' | 'yesterday';
};

const TIME_PRESETS: readonly TimePreset[] = [
  { labelKey: 'auth.login_log.time_preset.last_5_minutes', minutes: 5 },
  { labelKey: 'auth.login_log.time_preset.last_30_minutes', minutes: 30 },
  { labelKey: 'auth.login_log.time_preset.last_1_hour', hours: 1 },
  { labelKey: 'auth.login_log.time_preset.last_3_hours', hours: 3 },
  { labelKey: 'auth.login_log.time_preset.last_12_hours', hours: 12 },
  { labelKey: 'auth.login_log.time_preset.last_24_hours', hours: 24 },
  { labelKey: 'auth.login_log.time_preset.last_2_days', hours: 48 },
  { labelKey: 'auth.login_log.time_preset.last_7_days', hours: 24 * 7 },
  { labelKey: 'auth.login_log.time_preset.last_30_days', hours: 24 * 30 },
  { labelKey: 'auth.login_log.time_preset.today', preset: 'today' },
  { labelKey: 'auth.login_log.time_preset.yesterday', preset: 'yesterday' },
] as const;

const DURATION_PRESETS = TIME_PRESETS.filter((preset) => !preset.preset);
const CALENDAR_PRESETS = TIME_PRESETS.filter((preset) => preset.preset);

function buildTimePresetRange(preset: TimePreset): [dayjs.Dayjs, dayjs.Dayjs] {
  const now = dayjs();
  if (preset.preset === 'today') {
    return [now.startOf('day'), now.endOf('day')];
  }
  if (preset.preset === 'yesterday') {
    const yesterday = now.subtract(1, 'day');
    return [yesterday.startOf('day'), yesterday.endOf('day')];
  }
  if (preset.minutes) {
    return [now.subtract(preset.minutes, 'minute').startOf('minute'), now.endOf('minute')];
  }
  return [now.subtract(preset.hours || 0, 'hour').startOf('minute'), now.endOf('minute')];
}

const TRIGGER_PROPS = {
  className: 'time-range-filter__popup',
  autoAlignPopupWidth: false,
  popupAlign: { bottom: 6 },
  popupStyle: {
    // 弹层按内容自适应（双月日历 ~540px + 外壳），只约束不超出视口
    maxWidth: 'calc(100vw - 32px)',
    maxHeight: 640,
  },
} as const;

/**
 * 审计页共享的时间范围筛选器：快捷预设 + 双月日历 + 时分选择。
 * 空值表示不过滤；应用后通过 onChange 回传 'YYYY-MM-DD HH:mm' 起止串。
 */
const TimeRangeFilter: React.FC<TimeRangeFilterProps> = ({ value, onChange }) => {
  const { t } = useTranslation();
  const [popupVisible, setPopupVisible] = useState(false);
  const [draft, setDraft] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);
  const [presetLabelKey, setPresetLabelKey] = useState<string | null>(null);

  const rangeValue = useMemo<[dayjs.Dayjs, dayjs.Dayjs]>(() => {
    if (value.startedAt && value.endedAt) {
      const start = dayjs(value.startedAt);
      const end = dayjs(value.endedAt);
      if (start.isValid() && end.isValid()) {
        return [start, end];
      }
    }
    return [dayjs().subtract(24, 'hour').startOf('minute'), dayjs().endOf('minute')];
  }, [value.endedAt, value.startedAt]);

  const displayValue = draft ?? rangeValue;
  const hasValue = Boolean(value.startedAt && value.endedAt);
  const triggerLabel = hasValue
    ? t(presetLabelKey ?? 'auth.login_log.time_range.custom')
    : t('auth.login_log.time_range.label');

  const closePopup = useCallback(() => {
    setPopupVisible(false);
    setDraft(null);
  }, []);

  const applyRange = useCallback(
    (start: dayjs.Dayjs, end: dayjs.Dayjs, labelKey: string | null) => {
      setPresetLabelKey(labelKey);
      onChange({
        startedAt: start.format(TIME_RANGE_PAYLOAD_FORMAT),
        endedAt: end.format(TIME_RANGE_PAYLOAD_FORMAT),
      });
    },
    [onChange],
  );

  const handleShortcutApply = useCallback(
    (preset: TimePreset) => {
      const [start, end] = buildTimePresetRange(preset);
      applyRange(start, end, preset.labelKey);
      closePopup();
    },
    [applyRange, closePopup],
  );

  const handlePopupVisibleChange = useCallback(
    (visible?: boolean) => {
      const nextVisible = Boolean(visible);
      setPopupVisible(nextVisible);
      setDraft(nextVisible ? rangeValue : null);
    },
    [rangeValue],
  );

  const handleSelect = useCallback(
    (_dateStrings: string[], dateValues: dayjs.Dayjs[]) => {
      if (!dateValues || dateValues.length < 2) {
        return;
      }
      const nextStart = dateValues[0] ? dateValues[0].startOf('minute') : rangeValue[0];
      const nextEnd = dateValues[1] ? dateValues[1].endOf('minute') : rangeValue[1];
      setDraft([nextStart, nextEnd]);
    },
    [rangeValue],
  );

  const handleChange = useCallback(
    (_dateStrings: string[], dateValues: dayjs.Dayjs[]) => {
      if (!dateValues || dateValues.length < 2 || !dateValues[0] || !dateValues[1]) {
        return;
      }
      applyRange(dateValues[0].startOf('minute'), dateValues[1].endOf('minute'), null);
      setDraft(null);
    },
    [applyRange],
  );

  const renderShortcutRow = useCallback(
    (presets: readonly TimePreset[]) => (
      <div className="time-range-filter__shortcut-row">
        {presets.map((preset) => (
          <Button
            key={preset.labelKey}
            size="mini"
            type="outline"
            className={`time-range-filter__shortcut ${
              presetLabelKey === preset.labelKey && hasValue
                ? 'time-range-filter__shortcut--active'
                : ''
            }`}
            onClick={() => {
              handleShortcutApply(preset);
            }}
          >
            {t(preset.labelKey)}
          </Button>
        ))}
      </div>
    ),
    [handleShortcutApply, hasValue, presetLabelKey, t],
  );

  const renderPanel = useCallback(
    (panelNode: React.ReactNode) => {
      const [displayStart, displayEnd] = [
        displayValue[0].format(TIME_RANGE_FILTER_FORMAT),
        displayValue[1].format(TIME_RANGE_FILTER_FORMAT),
      ];

      return (
        <div className="time-range-filter__shell">
          <div className="time-range-filter__shortcuts">
            {renderShortcutRow(DURATION_PRESETS)}
            {renderShortcutRow(CALENDAR_PRESETS)}
          </div>

          <div className="time-range-filter__summary">
            <div className="time-range-filter__summary-item">
              <Typography.Text type="secondary" className="time-range-filter__summary-label">
                {t('auth.login_log.time_range.start')}
              </Typography.Text>
              <Typography.Text className="time-range-filter__summary-value">
                {displayStart}
              </Typography.Text>
            </div>
            <Typography.Text type="secondary" className="time-range-filter__summary-separator">
              {t('auth.login_log.time_range.to')}
            </Typography.Text>
            <div className="time-range-filter__summary-item time-range-filter__summary-item--end">
              <Typography.Text type="secondary" className="time-range-filter__summary-label">
                {t('auth.login_log.time_range.end')}
              </Typography.Text>
              <Typography.Text className="time-range-filter__summary-value">
                {displayEnd}
              </Typography.Text>
            </div>
          </div>

          <div className="time-range-filter__panel-body">{panelNode}</div>
        </div>
      );
    },
    [displayValue, renderShortcutRow, t],
  );

  return (
    <RangePicker
      allowClear={false}
      dayStartOfWeek={1}
      format={TIME_RANGE_FILTER_FORMAT}
      extra={
        <Button
          type="text"
          size="mini"
          onClick={() => {
            closePopup();
          }}
        >
          {t('common.cancel')}
        </Button>
      }
      showTime={{ format: 'HH:mm' }}
      value={rangeValue}
      popupVisible={popupVisible}
      position="bl"
      triggerElement={
        <Button className="time-range-filter__trigger" htmlType="button" type="outline">
          <span className="time-range-filter__trigger-label">{triggerLabel}</span>
          <IconCalendar className="time-range-filter__trigger-icon" />
        </Button>
      }
      triggerProps={TRIGGER_PROPS}
      panelRender={renderPanel}
      onChange={handleChange}
      onSelect={handleSelect}
      onVisibleChange={handlePopupVisibleChange}
    />
  );
};

export default TimeRangeFilter;

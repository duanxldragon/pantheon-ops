import React, { useMemo, useState } from 'react';
import { Button, Input, Tag } from '@arco-design/web-react';
import type { ContextSelectorOption, OperationalDataState } from './types';
import './operational.css';

export interface ContextSelectorProps {
  title: React.ReactNode;
  options: ContextSelectorOption[];
  selected: ContextSelectorOption[];
  excluded?: ContextSelectorOption[];
  labels: {
    searchPlaceholder: string;
    available: string;
    selected: string;
    add: React.ReactNode;
    remove: React.ReactNode;
    excluded: React.ReactNode;
    staleItem: React.ReactNode;
  } & Record<OperationalDataState, React.ReactNode>;
  state?: OperationalDataState;
  maxVisibleOptions?: number;
  onSelect?: (option: ContextSelectorOption) => void;
  onRemove?: (option: ContextSelectorOption) => void;
}
export default function ContextSelector({
  title,
  options,
  selected,
  excluded = [],
  labels,
  state = 'ready',
  maxVisibleOptions = 100,
  onSelect,
  onRemove,
}: Readonly<ContextSelectorProps>) {
  const [keyword, setKeyword] = useState('');
  const selectedIds = useMemo(
    () => new Set(selected.map((item) => `${item.source}:${item.id}`)),
    [selected],
  );
  const visibleOptions = options
    .filter((item) => String(item.label).toLowerCase().includes(keyword.toLowerCase()))
    .slice(0, Math.max(1, maxVisibleOptions));
  return (
    <section className="operational-primitive context-selector" aria-label={String(title)}>
      <div className="operational-primitive__header">
        <div className="operational-primitive__title">{title}</div>
        <Input.Search
          value={keyword}
          placeholder={labels.searchPlaceholder}
          onChange={setKeyword}
          allowClear
        />
      </div>
      {state !== 'ready' ? (
        <div className="operational-primitive__state">{labels[state]}</div>
      ) : (
        <div className="operational-primitive__body context-selector__layout">
          <div className="context-selector__column" aria-label={labels.available}>
            {visibleOptions.map((option) => (
              <div className="context-selector__option" key={`${option.source}:${option.id}`}>
                <span>
                  {option.label}{' '}
                  {option.stale ? <Tag color="orange">{labels.staleItem}</Tag> : null}
                </span>
                <Button
                  disabled={option.disabled || selectedIds.has(`${option.source}:${option.id}`)}
                  onClick={() => onSelect?.(option)}
                >
                  {labels.add}
                </Button>
              </div>
            ))}
          </div>
          <div className="context-selector__column" aria-label={labels.selected}>
            {selected.map((option) => (
              <div className="context-selector__option" key={`${option.source}:${option.id}`}>
                <span>{option.label}</span>
                <Button onClick={() => onRemove?.(option)}>{labels.remove}</Button>
              </div>
            ))}
            {excluded.length > 0 ? <Tag>{`${labels.excluded}: ${excluded.length}`}</Tag> : null}
          </div>
        </div>
      )}
    </section>
  );
}

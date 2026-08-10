import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Badge, Button, Card, Input, Popover } from '@arco-design/web-react';
import { IconClose, IconFilter, IconSearch } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';

const KEYWORD_DEBOUNCE_MS = 300;
const NARROW_QUERY = '(max-width: 768px)';

export interface SearchToolbarProps {
  /** 关键词当前值（受控）。传 undefined 表示该页不需要关键词框。 */
  keyword?: string;
  /** 关键词占位文案，如“按用户名、昵称或邮箱搜索”。 */
  keywordPlaceholder?: string;
  /** 关键词提交（防抖后或 Enter 触发）。 */
  onKeywordChange?: (keyword: string) => void;
  /** 行内高频筛选控件（Select 等，选完即查由调用方在 onChange 里触发）。 */
  inlineFilters?: React.ReactNode;
  /** 低频筛选控件，收进“筛选”弹层。 */
  advancedFilters?: React.ReactNode;
  /** 弹层内活跃筛选数量，用于徽标；由调用方根据查询状态计算。 */
  advancedActiveCount?: number;
  /** 是否存在任何活跃筛选（含关键词与行内），决定“清空”按钮显隐。 */
  hasActiveFilters?: boolean;
  /** 清空全部筛选。 */
  onClearAll?: () => void;
  /** 追加到工具栏尾部的自定义节点（如导出按钮）。 */
  extra?: React.ReactNode;
}

/**
 * 统一搜索工具栏：左侧关键词框 + 行内高频筛选 + 低频筛选弹层 + 条件清空入口。
 * 交互契约：所有筛选变更即时触发查询，不设“搜索”按钮；
 * 窄屏（<=768px）下行内筛选自动并入弹层。
 */
const SearchToolbar: React.FC<SearchToolbarProps> = ({
  keyword,
  keywordPlaceholder,
  onKeywordChange,
  inlineFilters,
  advancedFilters,
  advancedActiveCount = 0,
  hasActiveFilters = false,
  onClearAll,
  extra,
}) => {
  const { t } = useTranslation();
  const [draft, setDraft] = useState(keyword ?? '');
  const [syncedKeyword, setSyncedKeyword] = useState(keyword ?? '');
  const [popupVisible, setPopupVisible] = useState(false);
  const [narrow, setNarrow] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const committedRef = useRef(keyword ?? '');

  // 外部受控值变化时在渲染期同步草稿（避免 effect 级联渲染）。
  if ((keyword ?? '') !== syncedKeyword) {
    setSyncedKeyword(keyword ?? '');
    setDraft(keyword ?? '');
  }

  useEffect(() => {
    committedRef.current = keyword ?? '';
  }, [keyword]);

  useEffect(() => {
    if (globalThis.document === undefined) {
      return undefined;
    }
    const mediaQuery = globalThis.matchMedia(NARROW_QUERY);
    const sync = () => setNarrow(mediaQuery.matches);
    sync();
    mediaQuery.addEventListener('change', sync);
    return () => mediaQuery.removeEventListener('change', sync);
  }, []);

  useEffect(
    () => () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    },
    [],
  );

  const commitKeyword = useCallback(
    (value: string) => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
        debounceRef.current = null;
      }
      if (value === committedRef.current) {
        return;
      }
      committedRef.current = value;
      onKeywordChange?.(value);
    },
    [onKeywordChange],
  );

  const handleDraftChange = useCallback(
    (value: string) => {
      setDraft(value);
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
      debounceRef.current = setTimeout(() => commitKeyword(value), KEYWORD_DEBOUNCE_MS);
    },
    [commitKeyword],
  );

  const showKeyword = onKeywordChange !== undefined;
  const inlineInPopover = narrow && inlineFilters !== undefined && inlineFilters !== null;
  const popoverFilters = useMemo(() => {
    if (!inlineInPopover) {
      return advancedFilters;
    }
    if (advancedFilters === undefined || advancedFilters === null) {
      return inlineFilters;
    }
    return (
      <>
        {inlineFilters}
        {advancedFilters}
      </>
    );
  }, [advancedFilters, inlineFilters, inlineInPopover]);
  const hasPopover = popoverFilters !== undefined && popoverFilters !== null;

  return (
    <Card className="filter-panel search-toolbar" role="search" aria-label={t('common.filters')}>
      <div className="filter-panel__body search-toolbar__body">
        {showKeyword ? (
          <Input
            allowClear
            className="search-toolbar__keyword"
            prefix={<IconSearch />}
            placeholder={keywordPlaceholder ?? t('common.searchKeyword.placeholder')}
            value={draft}
            onChange={handleDraftChange}
            onPressEnter={() => commitKeyword(draft)}
            onClear={() => commitKeyword('')}
          />
        ) : null}
        {!inlineInPopover && inlineFilters ? (
          <div className="search-toolbar__inline">{inlineFilters}</div>
        ) : null}
        {hasPopover ? (
          <Popover
            trigger="click"
            position="br"
            popupVisible={popupVisible}
            onVisibleChange={setPopupVisible}
            content={<div className="search-toolbar__popover">{popoverFilters}</div>}
          >
            <Button
              className="search-toolbar__advanced-trigger"
              aria-expanded={popupVisible}
              aria-haspopup="dialog"
              icon={<IconFilter />}
            >
              {t('common.moreFilters')}
              {advancedActiveCount > 0 ? (
                <Badge count={advancedActiveCount} className="search-toolbar__badge" />
              ) : null}
            </Button>
          </Popover>
        ) : null}
        {hasActiveFilters && onClearAll ? (
          <Button
            type="text"
            className="search-toolbar__clear"
            icon={<IconClose />}
            onClick={onClearAll}
          >
            {t('common.clearFilters')}
          </Button>
        ) : null}
        {extra ? <div className="search-toolbar__extra">{extra}</div> : null}
      </div>
    </Card>
  );
};

export default SearchToolbar;

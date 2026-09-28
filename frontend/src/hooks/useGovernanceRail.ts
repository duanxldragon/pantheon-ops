import { useCallback, useState } from 'react';

interface UseGovernanceRailOptions {
  defaultExpanded?: boolean;
  enabled?: boolean;
}

export function useGovernanceRail(options?: UseGovernanceRailOptions) {
  const enabled = options?.enabled ?? true;
  const [expanded, setExpanded] = useState(options?.defaultExpanded ?? false);

  const open = useCallback(() => {
    if (enabled) {
      setExpanded(true);
    }
  }, [enabled]);
  const close = useCallback(() => setExpanded(false), []);
  const toggle = useCallback(() => {
    if (!enabled) {
      return;
    }
    setExpanded((current) => !current);
  }, [enabled]);

  return {
    expanded: enabled ? expanded : false,
    open,
    close,
    toggle,
  };
}

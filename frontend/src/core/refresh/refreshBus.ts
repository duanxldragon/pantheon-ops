import { useEffect, useMemo, useRef } from 'react';
import { getRefreshState } from './api';
import { hasAuthCookie } from '../../store/useAuthStore';
import { isLogoutTransitionActive } from '../../api/request';

export type PantheonRefreshTopic =
  | 'system:user:changed'
  | 'system:role:changed'
  | 'system:menu:changed'
  | 'system:dept:changed'
  | 'system:post:changed'
  | 'system:permission:changed'
  | 'system:dict:changed'
  | 'system:setting:changed'
  | 'system:i18n:changed';

interface RefreshPayload {
  topic: PantheonRefreshTopic;
  source?: string;
  at: number;
}

type RefreshHandler = (payload: RefreshPayload) => void;

const DEFAULT_REFRESH_POLL_INTERVAL_MS = 5000;

const REFRESH_EVENT_NAME = 'pantheon:refresh';
const REFRESH_CHANNEL_NAME = 'pantheon-refresh-bus';
const refreshEventTarget = new EventTarget();

let refreshChannel: BroadcastChannel | null = null;

function getRefreshChannel() {
  if (typeof globalThis.document === 'undefined' || typeof BroadcastChannel === 'undefined') {
    return null;
  }
  if (!refreshChannel) {
    refreshChannel = new BroadcastChannel(REFRESH_CHANNEL_NAME);
    refreshChannel.onmessage = (event: MessageEvent<RefreshPayload | RefreshPayload[]>) => {
      const payloads = Array.isArray(event.data) ? event.data : [event.data];
      payloads.forEach((payload) => {
        refreshEventTarget.dispatchEvent(
          new CustomEvent<RefreshPayload>(REFRESH_EVENT_NAME, { detail: payload }),
        );
      });
    };
  }
  return refreshChannel;
}

function normalizeTopics(topics: PantheonRefreshTopic | PantheonRefreshTopic[]) {
  return Array.isArray(topics) ? [...topics] : [topics];
}

export function publishRefresh(
  topics: PantheonRefreshTopic | PantheonRefreshTopic[],
  source?: string,
) {
  const payloads = normalizeTopics(topics).map<RefreshPayload>((topic) => ({
    topic,
    source,
    at: Date.now(),
  }));

  payloads.forEach((payload) => {
    refreshEventTarget.dispatchEvent(
      new CustomEvent<RefreshPayload>(REFRESH_EVENT_NAME, { detail: payload }),
    );
  });

  const channel = getRefreshChannel();
  if (channel) {
    channel.postMessage(payloads);
  }
}

export function subscribeRefresh(
  topics: PantheonRefreshTopic | PantheonRefreshTopic[],
  handler: RefreshHandler,
) {
  const topicSet = new Set(normalizeTopics(topics));
  const listener = (event: Event) => {
    const payload = (event as CustomEvent<RefreshPayload>).detail;
    if (!payload || !topicSet.has(payload.topic)) {
      return;
    }
    handler(payload);
  };

  refreshEventTarget.addEventListener(REFRESH_EVENT_NAME, listener);
  return () => {
    refreshEventTarget.removeEventListener(REFRESH_EVENT_NAME, listener);
  };
}

export function useRefreshSubscription(
  topics: PantheonRefreshTopic | PantheonRefreshTopic[],
  handler: RefreshHandler,
) {
  const handlerRef = useRef(handler);
  const normalizedTopics = useMemo(() => normalizeTopics(topics), [topics]);
  const topicKey = useMemo(() => normalizedTopics.join(','), [normalizedTopics]);

  useEffect(() => {
    handlerRef.current = handler;
  }, [handler]);

  useEffect(
    () =>
      subscribeRefresh(normalizedTopics, (payload) => {
        handlerRef.current(payload);
      }),
    [normalizedTopics, topicKey],
  );
}

export function useRefreshPolling(
  token: string | null,
  topics: PantheonRefreshTopic[],
  intervalMs = DEFAULT_REFRESH_POLL_INTERVAL_MS,
) {
  const versionsRef = useRef<Record<string, number>>({});
  const normalizedTopics = useMemo(() => normalizeTopics(topics).sort((a, b) => a.localeCompare(b)), [topics]);
  const topicKey = useMemo(() => normalizedTopics.join(','), [normalizedTopics]);
  const authToken = token || (typeof globalThis.document !== 'undefined' && hasAuthCookie() ? '_cookie' : null);

  useEffect(() => {
    versionsRef.current = {};
    if (!authToken || !topicKey) {
      return;
    }

    let active = true;
    let timer: number | null = null;

    const poll = async () => {
      if (isLogoutTransitionActive()) {
        return;
      }
      if (typeof globalThis.document !== 'undefined' && !hasAuthCookie()) {
        return;
      }
      try {
        const resp = await getRefreshState(normalizedTopics);
        if (!active) {
          return;
        }
        normalizedTopics.forEach((topic) => {
          const nextVersion = Number(resp.topics?.[topic] || 0);
          const previousVersion = Number(versionsRef.current[topic] || 0);
          if (previousVersion > 0 && nextVersion > previousVersion) {
            publishRefresh(topic, 'server/sync');
          }
          versionsRef.current[topic] = nextVersion;
        });
      } catch {
        return;
      }
    };

    void poll();
    timer = globalThis.setInterval(() => {
      void poll();
    }, intervalMs);

    return () => {
      active = false;
      if (timer) {
        globalThis.clearInterval(timer);
      }
    };
  }, [authToken, intervalMs, normalizedTopics, topicKey]);
}

import { apiRequest } from '../../api/request';
import type { PantheonRefreshTopic } from './refreshBus';

interface RefreshStateResp {
  topics: Record<string, number>;
}

export function getRefreshState(topics: PantheonRefreshTopic[]) {
  return apiRequest<RefreshStateResp>({
    url: '/system/refresh/state',
    method: 'get',
    params: {
      topics: topics.join(','),
    },
    skipErrorMessage: true,
  });
}

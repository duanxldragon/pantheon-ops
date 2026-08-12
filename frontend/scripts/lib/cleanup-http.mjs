function defaultSleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

export function getCleanupGetRetryDelayMs(attempt, phase = 'pre') {
  const baseDelay = phase === 'post' ? 1000 : 250;
  const maxDelay = phase === 'post' ? 5000 : 1000;
  return Math.min(baseDelay * (2 ** (attempt - 1)), maxDelay);
}

export function getCleanupGetRetryOptions(phase = 'pre') {
  return {
    phase,
    maxAttempts: phase === 'post' ? 6 : 3,
  };
}

export async function fetchCleanupJson(fetchImpl, url, options = {}) {
  const {
    path = new URL(url).pathname,
    headers,
    maxAttempts = 3,
    phase = 'pre',
    sleep = defaultSleep,
  } = options;

  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    const response = await fetchImpl(url, { headers });
    if (!response.ok) {
      if (response.status === 429 && attempt < maxAttempts) {
        await sleep(getCleanupGetRetryDelayMs(attempt, phase));
        continue;
      }
      throw new Error(`GET ${path} failed: HTTP ${response.status}`);
    }

    const payload = await response.json();
    if (payload.code !== 200) {
      throw new Error(`GET ${path} failed: code ${payload.code}`);
    }
    return payload.data;
  }

  throw new Error(`GET ${path} failed: retry budget exhausted`);
}

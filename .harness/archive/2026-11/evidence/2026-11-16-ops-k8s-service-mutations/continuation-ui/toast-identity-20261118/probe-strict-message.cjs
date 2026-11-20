const fs = require('node:fs');
const path = require('node:path');
const { createRequire } = require('node:module');
const frontendRequire = createRequire(path.resolve(process.cwd(), 'frontend/package.json'));
const { JSDOM } = frontendRequire('jsdom');
const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://127.0.0.1/', pretendToBeVisual: true });
for (const name of ['window', 'document', 'navigator', 'HTMLElement', 'SVGElement', 'Element', 'Node', 'MutationObserver']) {
  Object.defineProperty(globalThis, name, { configurable: true, value: dom.window[name] });
}
globalThis.getComputedStyle = dom.window.getComputedStyle.bind(dom.window);
globalThis.requestAnimationFrame = dom.window.requestAnimationFrame.bind(dom.window);
globalThis.cancelAnimationFrame = dom.window.cancelAnimationFrame.bind(dom.window);
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const React = frontendRequire('react');
const { createRoot } = frontendRequire('react-dom/client');
const Message = frontendRequire('@arco-design/web-react/lib/Message').default;
const warnings = [];
const originalError = console.error;
console.error = (...args) => warnings.push(args.map(String).join(' '));

async function probe(withId) {
  const container = document.createElement('div');
  document.body.appendChild(container);
  let api;
  let calls = 0;
  function Harness() {
    const [currentApi, holder] = Message.useMessage({ maxCount: 1 });
    api = currentApi;
    return holder;
  }
  const root = createRoot(container);
  await React.act(async () => root.render(React.createElement(React.StrictMode, null, React.createElement(Harness))));
  await React.act(async () => {
    calls++;
    api.error({ content: '服务已变化，请刷新列表并重新核对后操作。', duration: 0, ...(withId ? { id: 'c04-service-write-error' } : {}) });
  });
  const notices = [...container.querySelectorAll('.arco-message-error')].map((node) => ({ className: node.className, text: node.textContent, parentClass: node.parentElement.className }));
  const result = { withId, calls, noticeCount: notices.length, notices };
  if (withId) {
    await React.act(async () => api.error({ id: 'c04-service-write-error', content: '第二次错误', duration: 0 }));
    result.sameIdUpdateCount = container.querySelectorAll('.arco-message-error').length;
    result.sameIdUpdatedText = container.querySelector('.arco-message-error')?.textContent;
  }
  await React.act(async () => root.unmount());
  result.afterUnmountCount = container.querySelectorAll('.arco-message-error').length;
  container.remove();
  return result;
}

(async () => {
  const results = [await probe(false), await probe(true)];
  const receipt = { reactVersion: React.version, strictMode: true, actualInstalledMessage: true, results, warnings };
  fs.writeFileSync(path.join(__dirname, 'strict-message-probe.json'), JSON.stringify(receipt, null, 2) + '\n');
  console.error = originalError;
  console.log(JSON.stringify(receipt));
  if (results[0].calls !== 1 || results[0].noticeCount !== 2 || results[1].calls !== 1 || results[1].noticeCount !== 1
    || results[1].sameIdUpdateCount !== 1 || results[1].sameIdUpdatedText !== '第二次错误'
    || results.some((result) => result.afterUnmountCount !== 0)) process.exitCode = 1;
  dom.window.close();
})().catch((error) => { console.error = originalError; console.error(error); dom.window.close(); process.exitCode = 1; });

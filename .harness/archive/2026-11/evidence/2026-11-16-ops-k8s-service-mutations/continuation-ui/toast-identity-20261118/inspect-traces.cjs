const fs = require('node:fs');
const path = require('node:path');
const { createRequire } = require('node:module');
const { createHash } = require('node:crypto');
const frontendRequire = createRequire(path.resolve(process.cwd(), 'frontend/package.json'));
const JSZip = frontendRequire('jszip');
const root = '.harness/evidence/2026-11-17-ops-history-audit/continuation-c04-ui-toast-visibility-fixed-artifacts';
const cases = [
  'smoke-business-k8s-k8s-pag-9557e--draft-and-original-version',
  'smoke-business-k8s-k8s-pag-636ae-ed-primary-states-at-1440px',
  'smoke-business-k8s-k8s-pag-f798f-red-primary-states-at-390px',
];
function elements(node, parents = [], result = []) {
  if (!Array.isArray(node)) return result;
  if (typeof node[0] === 'string' && node[1] && typeof node[1] === 'object' && !Array.isArray(node[1])) {
    const item = { tag: node[0], className: node[1].class ?? '', parentClasses: parents, text: JSON.stringify(node).includes('服务已变化，请刷新列表并重新核对后操作。') };
    result.push(item);
    for (const child of node.slice(2)) elements(child, [...parents, item.className], result);
  } else for (const child of node) elements(child, parents, result);
  return result;
}
(async () => {
  const receipts = [];
  for (const scenario of cases) {
    const tracePath = `${root}/${scenario}/trace.zip`;
    const bytes = fs.readFileSync(tracePath);
    const zip = await JSZip.loadAsync(bytes);
    const snapshots = [], writes = [];
    for (const name of Object.keys(zip.files).filter((name) => name.endsWith('.trace') || name.endsWith('.network'))) {
      const lines = (await zip.file(name).async('string')).split('\n').filter(Boolean).map((line) => JSON.parse(line));
      for (const row of lines) {
        if (row.type === 'frame-snapshot') {
          const dom = elements(row.snapshot.html);
          const notices = dom.filter((element) => element.className.split(/\s+/).includes('arco-message-error'));
          if (notices.length === 2 && notices.every((notice) => notice.text)) snapshots.push({ callId: row.snapshot.callId, timestamp: row.snapshot.timestamp, noticeCount: notices.length, wrapperCount: dom.filter((element) => element.className.split(/\s+/).includes('arco-message-wrapper')).length, notices });
        }
        const request = row.snapshot?.request;
        if (request?.method === 'PUT' && request.url?.includes('/services/c04-routing')) writes.push({ method: request.method, url: request.url });
      }
    }
    if (!snapshots.length || writes.length !== 1) throw new Error(`Unexpected trace evidence: ${scenario}`);
    receipts.push({ scenario, tracePath, sha256: createHash('sha256').update(bytes).digest('hex'), writes, firstDuplicateSnapshot: snapshots[0] });
  }
  fs.writeFileSync(path.join(__dirname, 'trace-message-structure.json'), JSON.stringify(receipts, null, 2) + '\n');
  console.log(JSON.stringify(receipts.map((receipt) => ({ scenario: receipt.scenario, writes: receipt.writes.length, notices: receipt.firstDuplicateSnapshot.noticeCount, wrappers: receipt.firstDuplicateSnapshot.wrapperCount, callId: receipt.firstDuplicateSnapshot.callId }))));
})().catch((error) => { console.error(error); process.exitCode = 1; });

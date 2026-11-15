"""Root-run finite Argo product acceptance; credentials remain remote in memory."""
from pathlib import Path
import argparse
import ast
import hashlib
import json
import logging
import os
import secrets
import subprocess
import threading
import time
import tempfile
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ROOT = Path(__file__).resolve().parents[2]
EV = ROOT / '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
COMMAND = ['go', 'test', '-mod=readonly', '-race', '-count=1', '-timeout', '6m', '-run', '^TestArgoFinalProductLive$', '-v', './modules/business/cicd/argocd']


def source_hashes():
    return {p.relative_to(ROOT).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
            for p in sorted((ROOT / 'backend/modules/business/cicd/argocd').glob('*.go'))}


def remote_source():
    # Reuse the reviewed smart Git protocol and UID/label cleanup, not the old
    # provider-only test loop. AST extraction does not execute that driver.
    tree = ast.parse((ROOT / '.local/ops-acceptance/live-k8s-argo-provider-v7.py').read_text(encoding='utf-8'))
    original = next(ast.literal_eval(n.value) for n in tree.body if isinstance(n, ast.Assign)
                    and any(isinstance(t, ast.Name) and t.id == 'SCRIPT' for t in n.targets))
    pre = original[:original.index(' policy=[')]
    cleanup = original[original.index('except Exception as e:\n code=1;'):]
    product = r'''
 import re
 from urllib.parse import parse_qs, urlsplit
 def validate_request(method, path, body, *, application, namespace, project, uid, revisions):
     base = '/api/v1/applications/' + application
     parsed = urlsplit(path)
     if parsed.scheme or parsed.netloc or parsed.fragment:
         raise ValueError('relative-finite-provider-path-required')
     query = parse_qs(parsed.query, keep_blank_values=True)
     if method == 'GET':
         if body is not None:
             raise ValueError('no-get-body')
         if path == base or path == base + '/managed-resources':
             return False
         if parsed.path == base + '/manifests' and query == {'revision': [query.get('revision', [''])[0]]} and query['revision'][0] in revisions:
             return False
         if parsed.path == base + '/reviewed-actions/capabilities' and query == {'appNamespace': [namespace], 'project': [project]}:
             return False
         raise ValueError('unapproved-provider-read')
     if method != 'POST' or parsed.query or parsed.path not in [base + '/reviewed-actions/sync', base + '/reviewed-actions/rollback']:
         raise ValueError('only-reviewed-native-actions')
     rollback = parsed.path.endswith('/rollback')
     expected = {'name', 'appNamespace', 'project', 'prune', 'reviewed', 'id' if rollback else 'revision'}
     if not isinstance(body, dict) or set(body) != expected or body['name'] != application or body['appNamespace'] != namespace or body['project'] != project or body['prune'] is not False:
         raise ValueError('finite-owned-action-fields')
     fields = body['reviewed']
     if not isinstance(fields, dict) or set(fields) != {'expectedUID', 'expectedResourceVersion', 'resolvedRevision', 'actionMarker'}:
         raise ValueError('required-reviewed-fields')
     version = fields['expectedResourceVersion']
     marker = fields['actionMarker']
     if fields['expectedUID'] != uid or not isinstance(version, str) or not re.fullmatch(r'[0-9]{1,128}', version) or fields['resolvedRevision'] not in revisions or not isinstance(marker, str) or not re.fullmatch(r'[A-Za-z0-9_-]{1,128}', marker):
         raise ValueError('reviewed-identity-revision-marker')
     if rollback:
         if type(body['id']) is not int or body['id'] < 0 or body['id'] > (1 << 63) - 1:
             raise ValueError('immutable-native-history-id')
     elif body['revision'] != fields['resolvedRevision']:
         raise ValueError('same-resolved-sync-revision')
     return True
 policy=[f'p, proj:{PROJECT}:acceptance, applications, {action}, {PROJECT}/{APP}, allow' for action in ['get','sync']]
 create({'apiVersion':'argoproj.io/v1alpha1','kind':'AppProject','metadata':{'name':PROJECT,'namespace':ARGO,'labels':LABELS},'spec':{'sourceRepos':[REPO],'destinations':[{'server':'https://kubernetes.default.svc','namespace':NS}],'clusterResourceWhitelist':[],'namespaceResourceWhitelist':[{'group':'','kind':'Service'}],'roles':[{'name':'acceptance','policies':policy}]}})
 status,data=api('POST',f'projects/{PROJECT}/roles/acceptance/token',admin,{'expiresIn':900})
 check('finite exact application project token created',status==200 and bool(data.get('token')),expiresIn=900)
 limited=data.pop('token');data.clear()
 claims=json.loads(base64.urlsafe_b64decode(limited.split('.')[1]+'==='));iat=claims['iat']
 assert 0 < claims['exp']-iat <= 900
 app={'apiVersion':'argoproj.io/v1alpha1','kind':'Application','metadata':{'name':APP,'namespace':ARGO,'labels':LABELS},'spec':{'project':PROJECT,'destination':{'server':'https://kubernetes.default.svc','namespace':NS},'source':{'repoURL':REPO,'path':'.','targetRevision':OLD,'directory':{'include':'routing.json'}}}}
 status,data=api('POST','applications',admin,app)
 check('owned isolated Service-only application created',status==200)
 owned['Application']={'name':APP,'namespace':ARGO,'uid':data['metadata']['uid']}
 emit({'phase':'product-fixture-ready','Repo':REPO,'Namespace':NS,'Project':PROJECT,'Application':APP,'OldRevision':OLD,'NewRevision':NEW,'Token':limited,'credentialLifetimeSeconds':900,'certificateVerified':True,'providerAccess':'private explicit fixture via pinned SSH','authorityScope':'fixture-only; Base current identity authority remains a dependency'})
 mutations=0;tokenRevoked=False
 import sys
 for line in sys.stdin:
  request=json.loads(line)
  if request.get('command')=='close':break
  method=request.get('method');path=request.get('path','')
  if path=='/fixture/service' and method=='GET':
   live=json.loads(k('-n',NS,'get','service','argo-routing','-o','json'));assert live['metadata']['labels'].get('pantheon-p0-owner')==LABELS['pantheon-p0-owner']
   emit({'phase':'rpc','status':200,'data':{'revision':live['spec']['selector'].get('revision'),'uid':live['metadata']['uid']}});continue
  if path=='/fixture/counters' and method=='GET':
   emit({'phase':'rpc','status':200,'data':{'mutations':mutations}});continue
  if path=='/fixture/revoke-token' and method=='POST':
   status,_=api('DELETE',f'projects/{PROJECT}/roles/acceptance/token/{iat}',admin)
   tokenRevoked=status==200
   emit({'phase':'rpc','status':status,'data':{}});continue
  assert request.get('authorization')=='Bearer '+limited
  mutation=validate_request(method,path,request.get('body'),application=APP,namespace=ARGO,project=PROJECT,uid=owned['Application']['uid'],revisions=(OLD,NEW))
  if mutation:mutations+=1
  status,data=api(method,path[len('/api/v1/'):],limited,request.get('body'))
  # Raw provider responses stay in the private RPC channel. Public evidence
  # consists only of check names/statuses, never these payloads or credentials.
  emit({'phase':'rpc','status':status,'data':data})
 status=200
 if not tokenRevoked:status,_=api('DELETE',f'projects/{PROJECT}/roles/acceptance/token/{iat}',admin)
 check('finite provider credential revoked after product phase',status==200)
'''
    script = (pre + product + cleanup).replace('RUN_VALUE', repr(secrets.token_hex(3)))
    script = script.replace("'live-k8s-argo'", "'argo-final'").replace('live-k8s-argo-repo-', 'argo-final-repo-').replace('live-k8s-argo-smart-http.py', 'argo-final-smart-http.py')
    # Remove fixture files only after server identity/termination and after
    # resolved path/owner/known commit checks. No broad remote deletion.
    tail = " emit({'phase':'result','exitCode':code or cleanup_code"
    assert tail in script
    extra = r'''
 if git_server and git_server.poll() is not None and cleanup_code==0:
  try:
   import shutil
   resolved=fixture.resolve();assert resolved.parent==Path('/opt/pantheon-p0-20261109').resolve() and resolved.name=='argo-final-repo-'+RUN
   assert (bare/'refs/heads/main').read_text().strip()==NEW
   for kind in ['Application','AppProject','Namespace']:
    identity=owned.get(kind)
    if identity:
     args=['-n',identity['namespace']] if identity['namespace'] else []
     assert not k(*args,'get',kind,identity['name'],'--ignore-not-found','-o','name').strip()
   shutil.rmtree(resolved)
   emit({'phase':'owned-fixture-files-cleanup','resolvedPathChecked':True,'commitChecked':True,'serverAbsent':True,'exitCode':0})
  except Exception as e:cleanup_code=1;emit({'phase':'owned-fixture-files-cleanup','exitCode':1,'errorType':type(e).__name__})
 try:
  pods=json.loads(k('-n',ARGO,'get','pods','-o','json'))['items']
  assert len(pods)==4
  cap=0
  for pod in pods:
   assert any(c['type']=='Ready' and c['status']=='True' for c in pod['status']['conditions'])
   for container in pod['spec']['containers']:
    limit=container['resources']['limits']['memory'];assert limit.endswith('Mi');cap+=int(limit[:-2])
   for state in pod['status'].get('containerStatuses',[]):
    assert state.get('state',{}).get('terminated',{}).get('reason')!='OOMKilled' and state.get('lastState',{}).get('terminated',{}).get('reason')!='OOMKilled'
  assert cap==960
  emit({'phase':'retained-provider-health','readyCorePods':4,'memoryLimitMi':cap,'oomObserved':False,'exitCode':0})
 except Exception as e:cleanup_code=1;emit({'phase':'retained-provider-health','exitCode':1,'errorType':type(e).__name__})
'''
    script = script.replace(tail, extra + tail)
    before = "obj=json.loads(k(*args,'get',kind,identity['name'],'-o','json'))"
    assert script.count(before) == 1
    script = script.replace(before, before.replace('obj=', 'cleanup_obj='), 1)
    before = "assert obj['metadata']['uid']==identity['uid'] and obj['metadata']['labels'].get('pantheon-p0-owner')=='ops-p0-20261109' and obj['metadata']['labels'].get('pantheon-p0-driver')=='argo-final'"
    assert script.count(before) == 1
    script = script.replace(before, before.replace('obj[', 'cleanup_obj['), 1)
    before = "checks=0;stage='initialize';owned={};admin=None;code=0;git_server=None"
    assert script.count(before) == 1
    script = script.replace(before, before + ';bare=None', 1)
    before = "   assert (bare/'refs/heads/main').read_text().strip()==NEW"
    assert script.count(before) == 1
    script = script.replace(before, "   assert bare is not None\n" + before, 1)
    return script


def safe_schema(value):
    # Provider field names are allowlisted too: never enumerate map keys in
    # data, labels, annotations, selectors, messages or arbitrary manifests.
    paths = ["metadata", "metadata.uid", "metadata.resourceVersion", "spec", "spec.project", "spec.source", "spec.source.repoURL", "spec.source.path", "spec.source.targetRevision", "spec.source.directory", "spec.sources", "spec.destination", "spec.destination.server", "spec.destination.namespace", "spec.syncPolicy", "spec.syncPolicy.automated", "operation", "status", "status.history", "status.history[].id", "status.operationState", "status.operationState.startedAt", "status.operationState.finishedAt", "manifests", "manifests[]", "revision", "items", "items[].normalizedLiveState"]
    types = {}
    def walk(obj, parts):
        if not parts: return type(obj).__name__
        segment = parts[0]; is_array = segment.endswith("[]"); key = segment[:-2] if is_array else segment
        if not isinstance(obj, dict) or key not in obj: return "missing"
        child = obj[key]
        if is_array:
            if not isinstance(child, list): return type(child).__name__
            if not child: return "empty-array"
            return ",".join(sorted({walk(item, parts[1:]) for item in child[:200]}))
        return walk(child, parts[1:])
    for path in paths: types[path] = walk(value, path.split("."))
    resource_paths = ['apiVersion', 'kind', 'metadata', 'metadata.name', 'metadata.namespace', 'metadata.labels', 'metadata.annotations', 'spec', 'spec.ports', 'spec.selector', 'status']
    # Decode rendered/live JSON only in private memory. Return field types;
    # never traverse labels, annotations, selectors or application data.
    if isinstance(value, dict):
        arrays = [('manifests', value.get('manifests', []))]
        items = value.get('items', [])
        if isinstance(items, list):
            arrays.append(('items[].normalizedLiveState', [item.get('normalizedLiveState') for item in items[:200] if isinstance(item, dict)]))
        for prefix, array in arrays:
            if not isinstance(array, list): continue
            decoded = []
            for item in array[:200]:
                try: decoded.append(json.loads(item) if isinstance(item, str) else None)
                except (ValueError, UnicodeError): decoded.append(None)
            for path in resource_paths:
                types[prefix + '[decoded].' + path] = ','.join(sorted({walk(item, path.split('.')) for item in decoded})) if decoded else 'empty-array'
    return types

def safe_path(path):
    if path.startswith('/api/v1/applications/'):
        suffix = path.split('?', 1)[0].split('/')[5:]
        if not suffix: return '/api/v1/applications/{application}'
        if suffix == ['manifests']: return '/api/v1/applications/{application}/manifests'
        if suffix == ['managed-resources']: return '/api/v1/applications/{application}/managed-resources'
        if suffix in [['reviewed-actions', 'capabilities'], ['reviewed-actions', 'sync'], ['reviewed-actions', 'rollback']]: return '/api/v1/applications/{application}/' + '/'.join(suffix)
    if path in ['/fixture/service', '/fixture/counters', '/fixture/uncertain-next', '/fixture/revoke-token']: return path
    return '/{unrecognized}'

def run():
    from remote import connect  # Coordinating root's existing authorized adapter; final leaf only stages.
    logging.getLogger('paramiko.transport').setLevel(logging.CRITICAL)
    receipt = {'scope': 'ArgoCDService real provider integration; explicit fixture authority and private provider relay; no full Base auth/audit/public-origin claim',
               'host': '115.190.251.143', 'command': COMMAND, 'startedAt': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
               'sourceHashesBefore': source_hashes(), 'records': [], 'exitCode': 1}
    script = (ROOT / '.local/ops-acceptance/argo-reviewed-remote-staged-v4.py').read_text(encoding='utf-8')
    staged = json.loads((EV / 'argo-reviewed-phase-request-v4.json').read_text(encoding='utf-8'))
    if hashlib.sha256(script.encode()).hexdigest() != staged['remoteScriptSHA256']:
        raise RuntimeError('staged remote script hash changed; regenerate reviewable phase request')
    if hashlib.sha256(Path(__file__).read_bytes()).hexdigest() != staged['localScriptSHA256'] or receipt['sourceHashesBefore'] != staged['sourceHashes']:
        raise RuntimeError('reviewed diagnostic/product source hashes changed; root review required')
    process = server = None
    task_cache = None
    lock = threading.Lock()
    uncertain = {'next': False}
    started = time.monotonic()
    try:
        with connect(auth_timeout=45, banner_timeout=20) as ssh:
            stdin, stdout, stderr = ssh.exec_command('python3 -c "import json,sys; exec(json.loads(sys.stdin.readline()))"', timeout=500)
            # A JSON string is a valid Python string literal. Remote source has
            # no credentials; it resolves approved Kubernetes references only
            # after the exact namespace ownership check.
            stdin.write(json.dumps(script) + '\n'); stdin.flush()
            while True:
                line = stdout.readline()
                if not line:
                    raise RuntimeError('fixture did not become ready')
                record = json.loads(line)
                private_token = record.pop('Token', None)
                receipt['records'].append(record)
                print(json.dumps(record), flush=True)
                if record.get('phase') == 'product-fixture-ready':
                    fixture = {k: record[k] for k in ['Repo', 'Namespace', 'Project', 'Application', 'OldRevision', 'NewRevision']}
                    fixture['AuthToken'] = private_token
                    break

            class Relay(BaseHTTPRequestHandler):
                def log_message(self, *args):
                    pass

                def forward(self):
                    try:
                        with lock:
                            if self.path == '/fixture/uncertain-next' and self.command == 'POST':
                                uncertain['next'] = True
                                status, data = 200, {}
                            else:
                                length = int(self.headers.get('Content-Length', '0'))
                                if length > 4 * 1024 * 1024:
                                    raise RuntimeError('request budget')
                                body = json.loads(self.rfile.read(length)) if length else None
                                stdin.write(json.dumps({'method': self.command, 'path': self.path, 'body': body, 'authorization': self.headers.get('Authorization')}) + '\n'); stdin.flush()
                                reply = json.loads(stdout.readline())
                                if reply.get('phase') != 'rpc':
                                    raise RuntimeError('invalid fixture reply')
                                status, data = reply['status'], reply['data']
                                trace = {'method': self.command, 'pathTemplate': safe_path(self.path), 'status': status, 'fieldTypes': safe_schema(data)}
                                receipt.setdefault('providerTrace', []).append(trace)
                                if self.command == 'POST' and self.path.split('?', 1)[0].endswith(('/reviewed-actions/sync', '/reviewed-actions/rollback')) and status == 200 and uncertain['next']:
                                    uncertain['next'] = False
                                    status, data = 500, {}
                            raw = json.dumps(data).encode()
                            self.send_response(status); self.send_header('Content-Type', 'application/json'); self.send_header('Content-Length', str(len(raw))); self.end_headers(); self.wfile.write(raw)
                    except Exception:
                        self.send_error(502, 'fixture transport unavailable')

                do_GET = do_POST = forward

            server = ThreadingHTTPServer(('127.0.0.1', 0), Relay)
            thread = threading.Thread(target=server.serve_forever, daemon=True); thread.start()
            fixture['Endpoint'] = 'http://127.0.0.1:' + str(server.server_port)
            task_cache = tempfile.TemporaryDirectory(prefix='argo-final-live-cache-', dir=ROOT / '.local/ops-acceptance')
            cache_path = Path(task_cache.name).resolve()
            if cache_path.parent != (ROOT / '.local/ops-acceptance').resolve() or not cache_path.name.startswith('argo-final-live-cache-'):
                raise RuntimeError('unexpected final owned cache path')
            env = os.environ.copy()
            env['GOCACHE'] = str(cache_path)
            env.update(PATH='D:/msys64/mingw64/bin;' + env['PATH'], CGO_ENABLED='1',
                       PANTHEON_TEST_DSN='root@tcp(127.0.0.1:33316)/ops_p0_acceptance?parseTime=true&loc=Local', PANTHEON_TEST_DSN_REQUIRED='1',
                       PANTHEON_ARGO_PRODUCT_FIXTURE=json.dumps(fixture))
            with (EV / 'argo-reviewed-live-v4-go.txt').open('x', encoding='utf-8') as out:
                process = subprocess.Popen(COMMAND, cwd=ROOT / 'backend', env=env, stdout=out, stderr=subprocess.STDOUT)
                try:
                    product_exit = process.wait(timeout=420)
                except subprocess.TimeoutExpired:
                    process.kill(); product_exit = process.wait(timeout=10)
                    receipt['productTimeout'] = True
            receipt['productExitCode'] = product_exit
            server.shutdown(); server.server_close(); server = None
            stdin.write(json.dumps({'command': 'close'}) + '\n'); stdin.flush(); stdin.channel.shutdown_write()
            for line in stdout:
                record = json.loads(line)
                if record.get('phase') == 'rpc':
                    raise RuntimeError('unexpected private response')
                receipt['records'].append(record); print(json.dumps(record), flush=True)
            cleanup_exit = stdout.channel.recv_exit_status()
            receipt.update(remoteExitCode=cleanup_exit, stderrPresent=bool(stderr.read()), exitCode=product_exit or cleanup_exit)
    except Exception as exc:
        receipt.update(errorType=type(exc).__name__, localPermissionErrno=getattr(exc, 'errno', None))
        if getattr(exc, 'errno', None) == 13:
            receipt['blockedBy'] = 'local SSH execution restriction; root must run staged script with authorized tool approval; no retry or bypass'
    finally:
        if process is not None and process.poll() is None:
            process.kill(); process.wait(timeout=10)
        if server is not None:
            server.shutdown(); server.server_close()
        if task_cache is not None:
            task_cache.cleanup()
            receipt['isolatedCacheCleaned'] = not cache_path.exists()
        receipt.update(sourceHashesAfter=source_hashes(), elapsedSeconds=round(time.monotonic() - started, 3))
        receipt['sourceHashesStable'] = receipt['sourceHashesBefore'] == receipt['sourceHashesAfter']
        if not receipt['sourceHashesStable']:
            receipt['exitCode'] = 2
        (EV / 'argo-reviewed-live-v4-result.json').open('x', encoding='utf-8').write(json.dumps(receipt, indent=2) + '\n')
        print(json.dumps({k: v for k, v in receipt.items() if k not in ['records', 'sourceHashesBefore', 'sourceHashesAfter']}, indent=2))
    return receipt['exitCode']



def stage():
    script = remote_source()
    compile(script, 'argo-reviewed-remote-staged-v4.py', 'exec')
    remote_path = ROOT / '.local/ops-acceptance/argo-reviewed-remote-staged-v4.py'
    with remote_path.open('x', encoding='utf-8', newline='\n') as f:
        f.write(script)
    request = {'parentTask': '2026-11-07-ops-planned-task-closeout', 'phase': 'root-execution-required',
               'script': str(Path(__file__).resolve()), 'command': ['python', str(Path(__file__).resolve()), '--phase', 'run'],
               'remoteScript': str(remote_path), 'remoteScriptSHA256': hashlib.sha256(remote_path.read_bytes()).hexdigest(),
               'localScriptSHA256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(), 'sourceHashes': source_hashes(),
               'host': '115.190.251.143', 'providerNamespace': 'pantheon-p0-argocd-20261109',
               'bounds': {'timeoutSeconds':500, 'credentialLifetimeSeconds':900, 'retainedCoreMemoryMi':960,
                          'workloadPods':0, 'cleanup':'exact owned UID/labels and fixture path; provider retained'},
               'protocol': 'separate TestArgoFinalProductLive; owned same-ticket retry recovers Unknown/terminal and provider total remains four',
               'scope': 'fixture authority and private verified provider relay; no Base current-session/IAM/audit or public-origin proof',
               'runtimeGap': 'reviewed provider deployment/readiness and real terminal have not yet been validated by this stage',
               'permissionBoundary': 'root owns credentialed remote execution; this final leaf does not run or bypass SSH restrictions'}
    with (EV / 'argo-reviewed-phase-request-v4.json').open('x', encoding='utf-8') as f:
        json.dump(request,f,indent=2); f.write('\n')
    print(json.dumps({k:v for k,v in request.items() if k != 'sourceHashes'},indent=2))


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--phase', choices=['stage','run'], required=True)
    args = parser.parse_args()
    if args.phase == 'stage':
        stage()
    else:
        staged = json.loads((EV / 'argo-reviewed-phase-request-v4.json').read_text(encoding='utf-8'))
        if hashlib.sha256(Path(__file__).read_bytes()).hexdigest() != staged['localScriptSHA256'] or source_hashes() != staged['sourceHashes']:
            raise RuntimeError('reviewed source/driver hash changed; prepare a new reviewed final phase before any remote access')
        raise SystemExit(run())

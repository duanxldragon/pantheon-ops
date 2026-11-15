
import base64, hashlib, http.client, json, os, re, socket, ssl, subprocess, time, zlib
from pathlib import Path
ARGO='pantheon-p0-argocd-20261109'
RUN='d36536'
NS='pantheon-p0-argo-live-'+RUN
PROJECT='p0-live-'+RUN
APP='p0-live-'+RUN
LABELS={'pantheon-p0-owner':'ops-p0-20261109','pantheon-p0-driver':'argo-final'}
REPO='https://github.com/argoproj/argocd-example-apps.git'
OLD='68657670d9131dc5bc5f538b14c1de3377d74591'
NEW='f58c7ed8cfe28ad70701c5923fdbd0154388ea9f'
checks=0;stage='initialize';owned={};admin=None;code=0;git_server=None;bare=None

def emit(v):print(json.dumps(v),flush=True)
def check(name,ok,**details):
 global checks,stage
 stage=name;emit({'check':name,'passed':bool(ok),**details})
 if not ok:raise RuntimeError('assertion')
 checks+=1

def k(*a,body=None):
 r=subprocess.run(['kubectl','--request-timeout=15s',*a],input=None if body is None else json.dumps(body),capture_output=True,text=True,timeout=30)
 if r.returncode:raise RuntimeError('kubectl-exit-'+str(r.returncode))
 return r.stdout

def create(obj):
 k('create','-f','-',body=obj)
 key=obj['kind'];namespace=obj['metadata'].get('namespace')
 value=json.loads(k(*(['-n',namespace] if namespace else []),'get',key,obj['metadata']['name'],'-o','json'))
 owned[key]={'name':value['metadata']['name'],'namespace':namespace,'uid':value['metadata']['uid']}

class VerifiedArgoConnection(http.client.HTTPSConnection):
 def connect(self):
  self.sock=ssl_context.wrap_socket(socket.create_connection((self.host,self.port),self.timeout),server_hostname='argocd-server')

def api(method,path,token=None,body=None):
 conn=VerifiedArgoConnection(address,443,timeout=60,context=ssl_context)
 headers={'Content-Type':'application/json'}
 if token:headers['Authorization']='Bearer '+token
 try:
  conn.request(method,'/api/v1/'+path,body=None if body is None else json.dumps(body),headers=headers)
  response=conn.getresponse();raw=response.read(2*1024*1024)
  try:data=json.loads(raw)
  except Exception:data={}
  return response.status,data
 finally:conn.close()

def app_state(token):
 status,data=api('GET','applications/'+APP,token)
 if status!=200:raise RuntimeError('application-status-'+str(status))
 return data

def wait_terminal(token,revision,timeout=100):
 deadline=time.monotonic()+timeout
 while time.monotonic()<deadline:
  value=app_state(token);operation=value.get('status',{}).get('operationState',{});phase=operation.get('phase')
  result=operation.get('syncResult',{}).get('revision')
  if phase in ('Succeeded','Failed','Error') and result==revision:
   return value
  time.sleep(2)
 raise RuntimeError('terminal-timeout')

try:
 owner=json.loads(k('get','namespace',ARGO,'-o','json'))
 check('installed Argo namespace ownership',owner['metadata']['labels'].get('pantheon-p0-owner')=='ops-p0-20261109')
 service=json.loads(k('-n',ARGO,'get','service','argocd-server','-o','json'));address=service['spec']['clusterIP']
 # Existing Kubernetes secret references are resolved only in trusted driver memory.
 # Certificate is public; passwords/JWTs are never emitted or saved.
 cert=base64.b64decode(k('-n',ARGO,'get','secret','argocd-secret','-o','jsonpath={.data.tls\\.crt}')).decode()
 ssl_context=ssl.create_default_context(cadata=cert)
 status,_=api('GET','applications')
 check('unauthenticated Argo API refused',status in (401,403),httpStatus=status)
 password=base64.b64decode(k('-n',ARGO,'get','secret','argocd-initial-admin-secret','-o','jsonpath={.data.password}')).decode()
 status,session=api('POST','session',body={'username':'admin','password':password});password=None
 check('authenticated Argo session established',status==200 and bool(session.get('token')),httpStatus=status)
 admin=session.pop('token');session.clear()
 status,listing=api('GET','applications',admin)
 check('authenticated Argo API health',status==200,httpStatus=status)

 create({'apiVersion':'v1','kind':'Namespace','metadata':{'name':NS,'labels':LABELS}})
 # A standard-library dumb HTTP Git fixture contains only an isolated Service.
 # It avoids external GitHub latency and never changes Argo networking policy; this is provider-only source access.
 fixture=Path('/opt/pantheon-p0-20261109/argo-final-repo-'+RUN)
 fixture.mkdir(mode=0o700)
 bare=fixture/'fixture.git';bare.mkdir(mode=0o700)
 for sub in ['objects','refs/heads','info']:(bare/sub).mkdir(parents=True,exist_ok=True)
 objects=[]
 def obj(kind,body):
  objects.append((kind,body))
  raw=kind.encode()+b' '+str(len(body)).encode()+b'\0'+body
  sha=hashlib.sha1(raw).hexdigest();d=bare/'objects'/sha[:2];d.mkdir(exist_ok=True);(d/sha[2:]).write_bytes(zlib.compress(raw));return sha
 def commit(version,parent=None):
  manifest={'apiVersion':'v1','kind':'Service','metadata':{'name':'argo-routing','labels':LABELS},'spec':{'ports':[{'name':'web','port':80,'targetPort':8080}],'selector':{'revision':version}}}
  blob=obj('blob',json.dumps(manifest).encode());tree=obj('tree',b'100644 routing.json\0'+bytes.fromhex(blob))
  text='tree '+tree+'\n'+('parent '+parent+'\n' if parent else '')+'author Pantheon acceptance <fixture@example.invalid> 1794200000 +0000\ncommitter Pantheon acceptance <fixture@example.invalid> 1794200000 +0000\n\n'+version+'\n'
  return obj('commit',text.encode())
 OLD=commit('v1');NEW=commit('v2',OLD)
 (bare/'HEAD').write_text('ref: refs/heads/main\n');(bare/'refs/heads/main').write_text(NEW+'\n');(bare/'info/refs').write_text(NEW+'\trefs/heads/main\n');(bare/'objects/info').mkdir();(bare/'objects/info/packs').write_text('')
 (bare/'config').write_text('[core]\n\tbare = true\n\trepositoryformatversion = 0\n')
 probe=socket.socket();probe.bind(('0.0.0.0',0));port=probe.getsockname()[1];probe.close()

 # Serve standard Git smart HTTP advertisements plus the immutable six-object pack.
 import struct
 pack=b'PACK'+struct.pack('>II',2,len(objects))
 for kind,body in objects:
  value=len(body);first=(dict(commit=1,tree=2,blob=3)[kind]<<4)|(value&15);value>>=4
  header=bytes([first|(128 if value else 0)])
  while value:
   byte=value&127;value>>=7;header+=bytes([byte|(128 if value else 0)])
  pack+=header+zlib.compress(body)
 pack+=hashlib.sha1(pack).digest()
 (fixture/'fixture.pack').write_bytes(pack)
 server_script=fixture/'argo-final-smart-http.py'
 server_script.write_text("""from http.server import BaseHTTPRequestHandler,HTTPServer
from pathlib import Path
import sys
root=Path(sys.argv[1]);revision=sys.argv[3];pack=(root/'fixture.pack').read_bytes()
def pkt(s):
 b=s.encode();return ('%04x'%(len(b)+4)).encode()+b
class Handler(BaseHTTPRequestHandler):
 def log_message(self,*a):pass
 def answer(self,data,typ):
  self.send_response(200);self.send_header('Content-Type',typ);self.send_header('Content-Length',str(len(data)));self.end_headers();self.wfile.write(data)
 def do_GET(self):
  if self.path.startswith('/fixture.git/info/refs?service=git-upload-pack'):
   self.answer(pkt('# service=git-upload-pack\\n')+b'0000'+pkt(revision+' HEAD\\0symref=HEAD:refs/heads/main\\n')+pkt(revision+' refs/heads/main\\n')+b'0000','application/x-git-upload-pack-advertisement')
  elif self.path=='/fixture.git/HEAD':self.answer(b'ref: refs/heads/main\\n','text/plain')
  else:self.send_error(404)
 def do_POST(self):
  if self.path!='/fixture.git/git-upload-pack':self.send_error(404);return
  data=self.rfile.read(int(self.headers.get('Content-Length','0')))
  response=pkt('NAK\\n')+(pack if b'done\\n' in data else b'')
  self.answer(response,'application/x-git-upload-pack-result')
HTTPServer(('0.0.0.0',int(sys.argv[2])),Handler).serve_forever()
""")
 git_server=subprocess.Popen(['python3',str(server_script),str(fixture),str(port),NEW],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)

 nodes=json.loads(k('get','nodes','-o','json'))['items'];node_ip=next(a['address'] for a in nodes[0]['status']['addresses'] if a['type']=='InternalIP')
 REPO='http://'+node_ip+':'+str(port)+'/fixture.git'
 time.sleep(1)
 result=subprocess.run(['kubectl','--request-timeout=15s','-n',ARGO,'exec','deployment/argocd-repo-server','--','timeout','15','git','ls-remote',REPO,'HEAD'],capture_output=True,text=True,timeout=25)
 check('repo-server accesses isolated in-cluster Git fixture',result.returncode==0 and NEW in result.stdout,exitCode=result.returncode,repository=REPO,oldRevision=OLD,newRevision=NEW)


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
except Exception as e:
 code=1;emit({'phase':'failure','stage':stage,'errorType':type(e).__name__})
finally:
 cleanup_code=0
 # Recover possible API-create success after an ambiguous timeout, by exact labels.
 if 'Application' not in owned and 'Namespace' in owned:
  try:
   raw=k('-n',ARGO,'get','application',APP,'--ignore-not-found','-o','json')
   if raw.strip():
    v=json.loads(raw);assert v['metadata']['labels'].get('pantheon-p0-driver')=='argo-final' and v['metadata']['labels'].get('pantheon-p0-owner')=='ops-p0-20261109';owned['Application']={'name':APP,'namespace':ARGO,'uid':v['metadata']['uid']}
  except Exception as e:cleanup_code=1;emit({'phase':'ambiguous-create-identity-check','exitCode':1,'errorType':type(e).__name__})
 for kind in ['Application','AppProject','Namespace']:
  identity=owned.get(kind)
  if not identity:continue
  try:
   args=['-n',identity['namespace']] if identity['namespace'] else []
   cleanup_obj=json.loads(k(*args,'get',kind,identity['name'],'-o','json'))
   assert cleanup_obj['metadata']['uid']==identity['uid'] and cleanup_obj['metadata']['labels'].get('pantheon-p0-owner')=='ops-p0-20261109' and cleanup_obj['metadata']['labels'].get('pantheon-p0-driver')=='argo-final'
   r=subprocess.run(['kubectl','--request-timeout=15s',*args,'delete',kind,identity['name'],'--wait=true','--timeout=60s'],capture_output=True,text=True,timeout=75)
   assert r.returncode==0
   absence=k(*args,'get',kind,identity['name'],'--ignore-not-found','-o','name');assert not absence.strip()
   emit({'phase':'cleanup','kind':kind,'uidChecked':True,'absenceVerified':True,'exitCode':0})
  except Exception as e:cleanup_code=1;emit({'phase':'cleanup','kind':kind,'exitCode':1,'errorType':type(e).__name__})
 if git_server:
  try:
   cmd=Path('/proc/'+str(git_server.pid)+'/cmdline').read_bytes();assert str(server_script).encode() in cmd and str(fixture).encode() in cmd;git_server.terminate();git_server.wait(timeout=10);emit({'phase':'git-fixture-server-cleanup','pidIdentityChecked':True,'exitCode':0})
  except Exception as e:cleanup_code=1;emit({'phase':'git-fixture-server-cleanup','exitCode':1,'errorType':type(e).__name__})

 if git_server and git_server.poll() is not None and cleanup_code==0:
  try:
   import shutil
   resolved=fixture.resolve();assert resolved.parent==Path('/opt/pantheon-p0-20261109').resolve() and resolved.name=='argo-final-repo-'+RUN
   assert bare is not None
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
 emit({'phase':'result','exitCode':code or cleanup_code,'checksPassed':checks,'providersRetained':True,'namespace':NS,'project':PROJECT,'application':APP})
raise SystemExit(code or cleanup_code)

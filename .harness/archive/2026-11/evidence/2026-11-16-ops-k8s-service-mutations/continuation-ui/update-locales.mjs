import { readFileSync, writeFileSync } from 'node:fs';

// C04-only key updates; read current files to preserve other owners' changes.
const rows = [
  ['business.k8s.service.selectorUpdateHint', '修改类型、端口和选择器；其他配置保持原样。若服务已变化，请关闭后刷新列表并重新编辑。', 'Edit the type, ports and selector while keeping other settings. If the Service changes, close the editor, refresh the list and edit again.', '種類、ポート、セレクターを編集し、その他の設定を保持します。Service が変更された場合は、閉じて一覧を更新し、再編集してください。', '유형, 포트, 셀렉터를 수정하고 다른 설정은 유지합니다. Service가 변경되면 편집기를 닫고 목록을 새로 고침한 뒤 다시 편집하세요.', 'Modifiez le type, les ports et le sélecteur en conservant les autres paramètres. Si le Service change, fermez cet éditeur, actualisez la liste et recommencez.'],
  ['business.k8s.service.requiredFields', '请填写服务名称和命名空间。', 'Enter the Service name and namespace.', 'Service 名と名前空間を入力してください。', 'Service 이름과 네임스페이스를 입력하세요.', 'Saisissez le nom du Service et son espace de noms.'],
  ['business.k8s.service.selectorInvalid', '选择器格式无效或键重复，请使用 key=value 并以逗号分隔。', 'The selector is malformed or has duplicate keys. Use comma-separated key=value pairs.', 'セレクターの形式が無効かキーが重複しています。key=value をカンマで区切ってください。', '셀렉터 형식이 잘못되었거나 키가 중복됩니다. key=value 쌍을 쉼표로 구분하세요.', 'Le sélecteur est mal formé ou contient des clés en double. Séparez les paires key=value par des virgules.'],
  ['business.k8s.service.portsInvalid', '请检查端口范围、名称和目标端口；多个端口必须使用互不重复的名称。', 'Check port ranges, names and target ports. Multiple ports must have unique names.', 'ポート範囲、名前、宛先ポートを確認してください。複数ポートには一意の名前が必要です。', '포트 범위, 이름, 대상 포트를 확인하세요. 여러 포트에는 고유한 이름이 필요합니다.', 'Vérifiez les plages de ports, leurs noms et leurs ports cibles. Plusieurs ports doivent avoir des noms uniques.'],
  ['business.k8s.service.selectorPlaceholder', 'app=example, component=api', 'app=example, component=api', 'app=example, component=api', 'app=example, component=api', 'app=example, component=api'],
  ['business.k8s.service.selectorHint', '使用逗号分隔 key=value；留空表示不使用选择器。', 'Separate key=value pairs with commas. Leave blank for a selectorless Service.', 'key=value をカンマで区切ります。空欄の場合はセレクターを使用しません。', 'key=value 쌍을 쉼표로 구분하세요. 비워 두면 셀렉터를 사용하지 않습니다.', 'Séparez les paires key=value par des virgules. Laissez vide pour un Service sans sélecteur.'],
  ['business.k8s.service.portsHint', '目标端口可填写数字或容器端口名称，留空则使用服务端口。多个端口必须命名；节点端口留空可由集群分配。', 'Target ports accept numbers or container port names; blank uses the Service port. Name each port when using multiple ports. Leave node ports blank for cluster allocation.', '宛先ポートは数値またはコンテナのポート名です。空欄では Service ポートを使用します。複数ポートには名前が必要です。ノードポートは空欄で自動割り当てされます。', '대상 포트에는 숫자나 컨테이너 포트 이름을 입력하세요. 비워 두면 Service 포트를 사용합니다. 여러 포트에는 이름이 필요합니다. 노드 포트를 비워 두면 클러스터가 할당합니다.', 'Le port cible accepte un numéro ou un nom de port du conteneur ; vide, il reprend le port du Service. Nommez chaque port si vous en utilisez plusieurs. Laissez le port de nœud vide pour une attribution automatique.'],
  ['business.k8s.service.portNumber', '端口 {{number}}', 'Port {{number}}', 'ポート {{number}}', '포트 {{number}}', 'Port {{number}}'],
  ['business.k8s.service.removePortNumber', '移除端口 {{number}}', 'Remove port {{number}}', 'ポート {{number}} を削除', '포트 {{number}} 제거', 'Supprimer le port {{number}}'],
  ['business.k8s.service.removePort', '移除端口', 'Remove port', 'ポートを削除', '포트 제거', 'Supprimer le port'],
  ['business.k8s.service.addPort', '添加端口', 'Add port', 'ポートを追加', '포트 추가', 'Ajouter un port'],
  ['business.k8s.service.field.portName', '端口名称', 'Port name', 'ポート名', '포트 이름', 'Nom du port'],
  ['business.k8s.service.field.nodePort', '节点端口', 'Node port', 'ノードポート', '노드 포트', 'Port de nœud'],
  ['k8s.service.resource_version_conflict', '服务已变化，请刷新列表并重新核对后操作。', 'The Service has changed. Refresh the list and review it before trying again.', 'Service が変更されました。一覧を更新し、確認してから再試行してください。', 'Service가 변경되었습니다. 목록을 새로 고침하고 확인한 뒤 다시 시도하세요.', 'Le Service a changé. Actualisez la liste et vérifiez-le avant de réessayer.'],
  ['k8s.service.resource_version_required', '服务标识或资源版本不完整，请刷新列表后重试。', 'The Service identity or resource version is incomplete. Refresh the list and try again.', 'Service の識別情報またはリソースバージョンが不足しています。一覧を更新して再試行してください。', 'Service 식별자 또는 리소스 버전이 불완전합니다. 목록을 새로 고침한 뒤 다시 시도하세요.', 'L’identité du Service ou sa version est incomplète. Actualisez la liste et réessayez.'],
  ['k8s.service.update_failed', 'Kubernetes Service 更新失败', 'Failed to update the Kubernetes Service', 'Kubernetes Service の更新に失敗しました', 'Kubernetes Service 업데이트 실패', 'Échec de la mise à jour du Service Kubernetes'],
  ['k8s.service.audit.update', '更新 Kubernetes Service', 'Update Kubernetes Service', 'Kubernetes Service を更新', 'Kubernetes Service 업데이트', 'Mettre à jour le Service Kubernetes'],
  ['k8s.namespace.binding_invalid', '命名空间绑定与集群的业务域或环境不匹配。', 'The namespace binding does not match the cluster business scope or environment.', '名前空間のバインディングがクラスターの業務スコープまたは環境と一致しません。', '네임스페이스 바인딩이 클러스터의 업무 범위 또는 환경과 일치하지 않습니다.', 'Le rattachement de l’espace de noms ne correspond pas au périmètre métier ou à l’environnement du cluster.'],
];
for (const [index, locale] of ['zh-CN', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR'].entries()) {
  const path = `frontend/src/modules/business/v12locales/${locale}.json`;
  const data = JSON.parse(readFileSync(path, 'utf8'));
  for (const [key, ...translations] of rows) data[key] = translations[index];
  writeFileSync(path, `${JSON.stringify(data, null, 2)}\n`);
}
writeFileSync(new URL('./canonical-zh-en.json', import.meta.url), `${JSON.stringify(rows.map(([key, zh, en]) => ({ key, zh, en })), null, 2)}\n`);
console.log(`Updated ${rows.length} C04 keys in five canonical locales.`);

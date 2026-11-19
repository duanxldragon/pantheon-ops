import type { K8sServicePort, K8sServiceRow } from '../api';

interface PortDraft {
  name: string;
  port: number;
  targetPort: string;
  nodePort?: number;
  protocol: string;
}

export interface ServiceDraft {
  name: string;
  namespace: string;
  type: string;
  selector: string;
  ports: PortDraft[];
}

export const emptyServicePort = (): PortDraft => ({
  name: '',
  port: 80,
  targetPort: '8080',
  protocol: 'TCP',
});

export function createServiceDraft(row?: K8sServiceRow): ServiceDraft {
  return row
    ? {
        name: row.name,
        namespace: row.namespace,
        type: row.type,
        selector: Object.entries(row.selector ?? {})
          .map(([key, value]) => `${key}=${value}`)
          .join(', '),
        ports: row.ports.map((port) => ({
          ...port,
          name: port.name ?? '',
          targetPort: String(port.targetPort),
        })),
      }
    : {
        name: '',
        namespace: 'default',
        type: 'ClusterIP',
        selector: '',
        ports: [emptyServicePort()],
      };
}

export function serviceDraftPayload(draft: ServiceDraft):
  | {
      selector: Record<string, string>;
      ports: K8sServicePort[];
    }
  | string {
  if (!draft.name.trim() || !draft.namespace.trim()) return 'business.k8s.service.requiredFields';
  const selector: Record<string, string> = {};
  if (draft.selector.trim()) {
    for (const entry of draft.selector.split(',')) {
      const parts = entry.trim().split('=');
      const key = parts[0].trim();
      if (parts.length !== 2 || !key || Object.hasOwn(selector, key)) {
        return 'business.k8s.service.selectorInvalid';
      }
      // Define own keys so a literal __proto__ label cannot disappear during parsing.
      Object.defineProperty(selector, key, { value: parts[1].trim(), enumerable: true });
    }
  }
  const isPort = (value: number) => Number.isInteger(value) && value > 0 && value <= 65535;
  const isPortName = (value: string) =>
    value.length <= 15 && /^[a-z0-9](?:[-a-z0-9]*[a-z0-9])?$/.test(value) && /[a-z]/.test(value);
  const isServicePortName = (value: string) =>
    value.length <= 63 && /^[a-z0-9](?:[-a-z0-9]*[a-z0-9])?$/.test(value);
  const names = new Set<string>();
  const ports: K8sServicePort[] = [];
  if (!draft.ports.length) return 'business.k8s.service.portsInvalid';
  for (const port of draft.ports) {
    const name = port.name.trim();
    const target = port.targetPort.trim();
    const targetPort = !target ? port.port : /^\d+$/.test(target) ? Number(target) : target;
    if (
      !isPort(port.port) ||
      (typeof targetPort === 'number' ? !isPort(targetPort) : !isPortName(targetPort)) ||
      (port.nodePort !== undefined && port.nodePort !== 0 && !isPort(port.nodePort)) ||
      (name && (!isServicePortName(name) || names.has(name))) ||
      (draft.ports.length > 1 && !name)
    )
      return 'business.k8s.service.portsInvalid';
    if (name) names.add(name);
    ports.push({
      name: name || undefined,
      port: port.port,
      targetPort,
      nodePort: draft.type === 'ClusterIP' ? undefined : port.nodePort,
      protocol: port.protocol,
    });
  }
  return { selector, ports };
}

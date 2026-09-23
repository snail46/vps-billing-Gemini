import { InstanceState } from './status';

export interface Instance {
  id: string;
  subscription_id: string;
  node_id: string | null;
  provider_id: string | null;
  provider_instance_id: string | null;
  name: string;
  desired_state: InstanceState | string;
  observed_state: InstanceState | string;
  cpu_cores: number;
  memory_mb: number;
  disk_gb: number;
  traffic_limit_gb: number | null;
  bandwidth_mbps: number | null;
  image_id: string | null;
  primary_ipv4: string | null;
  primary_ipv6: string | null;
  last_synced_at: string | null;
  version: number;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export interface InstanceActionResponse {
  operation_id: string;
  operation?: any;
}

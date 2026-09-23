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
  root_password?: string;
}

export interface TrafficStats {
  instance_id: string;
  used_bytes: number;
  limit_bytes?: number;
  total_bytes?: number;
  used_gb?: number;
  limit_gb?: number;
  bandwidth_mbps?: number | null;
  reset_date?: string;
}

export interface PortForwardRule {
  id: string;
  protocol: string;
  public_port: number;
  guest_port: number;
  description: string;
}


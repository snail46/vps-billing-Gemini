export interface Provider {
  id: string;
  name: string;
  provider_type: string;
  endpoint?: string;
  status: string;
  version?: string;
  capabilities: Record<string, any>;
  last_health_check_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Node {
  id: string;
  provider_id: string;
  provider_node_id?: string;
  name: string;
  region: string;
  status: string;
  cpu_total: number;
  memory_total_mb: number;
  disk_total_gb: number;
  cpu_allocated: number;
  memory_allocated_mb: number;
  disk_allocated_gb: number;
  cpu_reserved: number;
  memory_reserved_mb: number;
  disk_reserved_gb: number;
  weight: number;
  capabilities: Record<string, any>;
  last_seen_at?: string;
  created_at: string;
}

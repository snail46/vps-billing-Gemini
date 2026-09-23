export type OrderStatus =
  | "pending"
  | "paid"
  | "fulfilling"
  | "fulfilled"
  | "cancelled"
  | "refund_pending"
  | "refunded";

export type PaymentStatus =
  | "pending"
  | "processing"
  | "succeeded"
  | "failed"
  | "refunded";

export type InvoiceStatus =
  | "pending"
  | "paid"
  | "cancelled"
  | "refunded";

export type LedgerDirection = "debit" | "credit";

export interface Plan {
  id: string;
  product_id: string;
  node_group_id?: string;
  slug: string;
  name_i18n: Record<string, string>;
  status: string;
  cpu_cores: number;
  memory_mb: number;
  disk_gb: number;
  traffic_gb?: number;
  bandwidth_mbps?: number;
  ipv4_count: number;
  ipv6_count: number;
  nat_port_count: number;
  virtualization: string;
  billing_cycle: string;
  price_minor: number;
  currency: string;
  stock_mode: string;
  created_at: string;
  updated_at: string;
}

export interface Product {
  id: string;
  slug: string;
  name_i18n: Record<string, string>;
  description_i18n: Record<string, string>;
  status: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
  plans?: Plan[];
}

export interface OrderItem {
  id: string;
  order_id: string;
  product_id: string;
  plan_id: string;
  quantity: number;
  unit_price_minor: number;
  total_minor: number;
  product_snapshot: Record<string, unknown>;
  plan_snapshot: Record<string, unknown>;
  created_at: string;
}

export interface Order {
  id: string;
  order_no: string;
  user_id: string;
  status: OrderStatus;
  subtotal_minor: number;
  discount_minor: number;
  total_minor: number;
  currency: string;
  paid_at?: string;
  created_at: string;
  updated_at: string;
  items?: OrderItem[];
}

export interface Payment {
  id: string;
  payment_no: string;
  order_id: string;
  gateway: string;
  gateway_payment_id?: string;
  status: PaymentStatus;
  amount_minor: number;
  currency: string;
  idempotency_key: string;
  paid_at?: string;
  created_at: string;
  updated_at: string;
}

export interface InvoiceItem {
  id: string;
  invoice_id: string;
  description_i18n: Record<string, string>;
  quantity: number;
  unit_amount_minor: number;
  total_minor: number;
  created_at: string;
}

export interface Invoice {
  id: string;
  invoice_no: string;
  user_id: string;
  subscription_id?: string;
  order_id?: string;
  status: InvoiceStatus;
  amount_minor: number;
  currency: string;
  due_at?: string;
  paid_at?: string;
  created_at: string;
  updated_at: string;
  items?: InvoiceItem[];
}

export interface Wallet {
  id: string;
  user_id: string;
  currency: string;
  available_balance_minor: number;
  created_at: string;
  updated_at: string;
}

export interface LedgerEntry {
  id: string;
  transaction_id: string;
  account_type: string;
  account_id: string;
  direction: LedgerDirection;
  amount_minor: number;
  currency: string;
  created_at: string;
}

export interface LedgerTransaction {
  id: string;
  type: string;
  reference_type?: string;
  reference_id?: string;
  description?: string;
  created_at: string;
  entries?: LedgerEntry[];
}

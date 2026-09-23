import { apiFetch } from "./client";
import { Product, Order, Invoice, Wallet, LedgerTransaction, Payment } from "../types/commerce";

export interface CreateOrderInput {
  items: {
    plan_id: string;
    quantity: number;
  }[];
}

export const commerceApi = {
  // Public
  async listProducts(): Promise<{ products: Product[] }> {
    const res = await apiFetch<{ products: Product[] }>("/api/v1/products");
    if (!res.success) throw new Error(res.error?.message_key || "failed to list products");
    return res.data;
  },

  // User Orders & Checkout
  async createOrder(input: CreateOrderInput): Promise<{ order: Order }> {
    const res = await apiFetch<{ order: Order }>("/api/v1/orders", {
      method: "POST",
      body: JSON.stringify(input),
    });
    if (!res.success) throw new Error(res.error?.message_key || "failed to create order");
    return res.data;
  },

  async listOrders(): Promise<{ orders: Order[] }> {
    const res = await apiFetch<{ orders: Order[] }>("/api/v1/orders");
    if (!res.success) throw new Error(res.error?.message_key || "failed to list orders");
    return res.data;
  },

  async payOrder(orderId: string, gateway = "fake"): Promise<{ payment: Payment; checkout_url: string }> {
    const res = await apiFetch<{ payment: Payment; checkout_url: string }>(`/api/v1/orders/${orderId}/pay`, {
      method: "POST",
      body: JSON.stringify({ gateway }),
    });
    if (!res.success) throw new Error(res.error?.message_key || "failed to initiate payment");
    return res.data;
  },

  async simulatePayment(paymentNo: string): Promise<unknown> {
    const res = await apiFetch("/api/v1/payments/fake/simulate", {
      method: "POST",
      body: JSON.stringify({ payment_no: paymentNo }),
    });
    if (!res.success) throw new Error(res.error?.message_key || "payment simulation failed");
    return res.data;
  },

  // User Wallet & Invoices
  async getWallet(): Promise<{ wallet: Wallet }> {
    const res = await apiFetch<{ wallet: Wallet }>("/api/v1/wallet");
    if (!res.success) throw new Error(res.error?.message_key || "failed to get wallet");
    return res.data;
  },

  async listInvoices(): Promise<{ invoices: Invoice[] }> {
    const res = await apiFetch<{ invoices: Invoice[] }>("/api/v1/invoices");
    if (!res.success) throw new Error(res.error?.message_key || "failed to list invoices");
    return res.data;
  },

  // Admin Commerce
  async adminListOrders(): Promise<{ orders: Order[] }> {
    const res = await apiFetch<{ orders: Order[] }>("/api/v1/admin/orders");
    if (!res.success) throw new Error(res.error?.message_key || "failed to list admin orders");
    return res.data;
  },

  async adminListInvoices(): Promise<{ invoices: Invoice[] }> {
    const res = await apiFetch<{ invoices: Invoice[] }>("/api/v1/admin/invoices");
    if (!res.success) throw new Error(res.error?.message_key || "failed to list admin invoices");
    return res.data;
  },

  async adminListLedger(): Promise<{ transactions: LedgerTransaction[] }> {
    const res = await apiFetch<{ transactions: LedgerTransaction[] }>("/api/v1/admin/ledger");
    if (!res.success) throw new Error(res.error?.message_key || "failed to list admin ledger");
    return res.data;
  },
};

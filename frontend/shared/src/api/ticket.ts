import { apiFetch } from './client';
import { Ticket, TicketMessage } from '../types/ticket';

export interface CreateTicketInput {
  subject: string;
  priority: 'low' | 'medium' | 'high';
  message: string;
}

export const ticketApi = {
  // User methods
  async listMyTickets(): Promise<Ticket[]> {
    const res = await apiFetch<{ tickets: Ticket[] }>('/api/v1/tickets');
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list tickets');
    return res.data?.tickets || [];
  },

  async createTicket(input: CreateTicketInput): Promise<Ticket> {
    const res = await apiFetch<{ ticket: Ticket }>('/api/v1/tickets', {
      method: 'POST',
      body: JSON.stringify(input),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to create ticket');
    return res.data!.ticket;
  },

  async getTicket(id: string): Promise<Ticket> {
    const res = await apiFetch<{ ticket: Ticket }>(`/api/v1/tickets/${id}`);
    if (!res.success) throw new Error(res.error?.message_key || 'failed to get ticket');
    return res.data!.ticket;
  },

  async userReplyTicket(id: string, message: string): Promise<TicketMessage> {
    const res = await apiFetch<{ message: TicketMessage }>(`/api/v1/tickets/${id}/reply`, {
      method: 'POST',
      body: JSON.stringify({ message }),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to reply ticket');
    return res.data!.message;
  },

  // Admin methods
  async adminListTickets(limit = 50, offset = 0): Promise<Ticket[]> {
    const res = await apiFetch<{ tickets: Ticket[] }>(`/api/v1/admin/tickets?limit=${limit}&offset=${offset}`);
    if (!res.success) throw new Error(res.error?.message_key || 'failed to list tickets');
    return res.data?.tickets || [];
  },

  async adminReplyTicket(id: string, message: string): Promise<TicketMessage> {
    const res = await apiFetch<{ message: TicketMessage }>(`/api/v1/admin/tickets/${id}/reply`, {
      method: 'POST',
      body: JSON.stringify({ message }),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to reply ticket');
    return res.data!.message;
  },

  async adminUpdateTicketStatus(id: string, status: 'open' | 'closed'): Promise<string> {
    const res = await apiFetch<{ status: string }>(`/api/v1/admin/tickets/${id}/status`, {
      method: 'POST',
      body: JSON.stringify({ status }),
    });
    if (!res.success) throw new Error(res.error?.message_key || 'failed to update ticket status');
    return res.data!.status;
  },
};

export interface TicketMessage {
  id: string;
  ticket_id: string;
  user_id?: string;
  admin_id?: string;
  sender_type: 'user' | 'admin';
  message: string;
  created_at: string;
}

export interface Ticket {
  id: string;
  user_id: string;
  subject: string;
  status: 'open' | 'closed';
  priority: 'low' | 'medium' | 'high';
  created_at: string;
  updated_at: string;
  messages?: TicketMessage[];
}

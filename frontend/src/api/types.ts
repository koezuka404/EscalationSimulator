export type User = {
  id: string;
  name: string;
  email: string;
  role: string;
  customer_id: string;
  status: string;
};

export type LoginResult = {
  access_token: string;
  expires_in: number;
  user: User;
};

export type Customer = {
  id: string;
  name: string;
  plan: string;
  sla_minutes: number;
};

export type MyTicket = {
  id: string;
  title: string;
  severity: number;
  status: string;
  created_at: string;
  remaining_minutes: number;
  overdue: boolean;
  overdue_minutes: number;
  assignee_name: string;
};

export type WaitingTicket = {
  rank: number;
  id: string;
  title: string;
  severity: number;
  plan: string;
  wait_minutes: number;
  priority_score: number;
  remaining_minutes: number;
  overdue: boolean;
  overdue_minutes: number;
  customer_name: string;
};

export type AgentPresence = {
  user_id: string;
  name: string;
  status: string;
};

export type WorkNote = {
  id: string;
  body: string;
  author_name: string;
  created_at: string;
};

export type SeverityChange = {
  from_severity: number;
  to_severity: number;
  reason: string;
  changed_by_name: string;
  created_at: string;
};

export type TicketDetail = {
  id: string;
  title: string;
  description: string;
  severity: number;
  category: string;
  customer_id: string;
  customer_name: string;
  plan: string;
  status: string;
  assignee_id: string;
  assignee_name: string;
  priority_score: number;
  created_at: string;
  claimed_at?: string;
  closed_at?: string;
  close_comment?: string;
  work_notes: WorkNote[];
  severity_changes: SeverityChange[];
};

export type CreatedTicket = {
  id: string;
  title: string;
  status: string;
  assignee_id: string;
};

export type AdminTicket = {
  id: string;
  title: string;
  severity: number;
  status: string;
  customer_id: string;
  customer_name: string;
  plan: string;
  assignee_id?: string;
  assignee_name?: string;
  priority_score: number;
  created_at: string;
  remaining_minutes: number;
  overdue: boolean;
  overdue_minutes: number;
};

export type DashboardNumbers = {
  waiting_count: number;
  in_progress_count: number;
  average_handle_minutes: number;
  average_first_response_minutes: number;
  overdue_count: number;
  overdue_24h_count: number;
  available_count: number;
  busy_count: number;
  offline_count: number;
};

export type DemoRun = {
  id: string;
  total_count: number;
  interval_seconds: number;
  distribution: string;
  status: string;
  generated_count: number;
};

export type LiveEvent = {
  type: string;
  tickets?: WaitingTicket[];
  agents?: AgentPresence[];
  dashboard?: DashboardNumbers;
  ticket_id?: string;
  assignee_id?: string;
  user_id?: string;
  name?: string;
  status?: string;
  generated_count?: number;
  total_count?: number;
  interval_seconds?: number;
  distribution?: string;
};

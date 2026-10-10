export const paths = {
  login: "/login",
  register: "/register",
  tickets: "/tickets",
  newTicket: "/tickets/new",
  ticket: (id: string) => `/tickets/${id}`,
  queue: "/agent/queue",
  agentTicket: (id: string) => `/agent/tickets/${id}`,
  dashboard: "/admin/dashboard",
  adminTickets: "/admin/tickets",
  adminTicket: (id: string) => `/admin/tickets/${id}`,
  customers: "/admin/customers",
  users: "/admin/users",
  demo: "/admin/demo",
};

export function homeFor(role: string): string {
  if (role === "agent") return paths.queue;
  if (role === "admin") return paths.dashboard;
  return paths.tickets;
}

import { Navigate, Route, Routes } from "react-router-dom";
import type { ReactNode } from "react";
import { useAuth } from "./auth/AuthProvider";
import { homeFor, paths } from "./paths";
import { LoginPage } from "./pages/LoginPage";
import { RegisterPage } from "./pages/RegisterPage";
import { MyTicketsPage } from "./pages/applicant/MyTicketsPage";
import { NewTicketPage } from "./pages/applicant/NewTicketPage";
import { TicketDetailPage as ApplicantTicketPage } from "./pages/applicant/TicketDetailPage";
import { QueuePage } from "./pages/agent/QueuePage";
import { TicketDetailPage as AgentTicketPage } from "./pages/agent/TicketDetailPage";
import { DashboardPage } from "./pages/admin/DashboardPage";
import { AllTicketsPage } from "./pages/admin/AllTicketsPage";
import { TicketDetailPage as AdminTicketPage } from "./pages/admin/TicketDetailPage";
import { CustomersPage } from "./pages/admin/CustomersPage";
import { UsersPage } from "./pages/admin/UsersPage";
import { DemoPage } from "./pages/admin/DemoPage";

export function App() {
  return (
    <Routes>
      <Route path={paths.login} element={<LoginPage />} />
      <Route path={paths.register} element={<RegisterPage />} />
      <Route path={paths.tickets} element={<Require role="applicant"><MyTicketsPage /></Require>} />
      <Route path={paths.newTicket} element={<Require role="applicant"><NewTicketPage /></Require>} />
      <Route path="/tickets/:ticketId" element={<Require role="applicant"><ApplicantTicketPage /></Require>} />
      <Route path={paths.queue} element={<Require role="agent"><QueuePage /></Require>} />
      <Route path="/agent/tickets/:ticketId" element={<Require role="agent"><AgentTicketPage /></Require>} />
      <Route path={paths.dashboard} element={<Require role="admin"><DashboardPage /></Require>} />
      <Route path={paths.adminTickets} element={<Require role="admin"><AllTicketsPage /></Require>} />
      <Route path="/admin/tickets/:ticketId" element={<Require role="admin"><AdminTicketPage /></Require>} />
      <Route path={paths.customers} element={<Require role="admin"><CustomersPage /></Require>} />
      <Route path={paths.users} element={<Require role="admin"><UsersPage /></Require>} />
      <Route path={paths.demo} element={<Require role="admin"><DemoPage /></Require>} />
      <Route path="*" element={<HomeRedirect />} />
    </Routes>
  );
}

function Require({ role, children }: { role: string; children: ReactNode }) {
  const { user, ready } = useAuth();
  if (!ready) return <p className="p-8 text-sm text-stone-600">ログインを確認しています。</p>;
  if (!user) return <Navigate to={paths.login} replace />;
  if (user.role !== role) return <Navigate to={homeFor(user.role)} replace />;
  return children;
}

function HomeRedirect() {
  const { user, ready } = useAuth();
  if (!ready) return <p className="p-8 text-sm text-stone-600">ログインを確認しています。</p>;
  if (!user) return <Navigate to={paths.login} replace />;
  return <Navigate to={homeFor(user.role)} replace />;
}

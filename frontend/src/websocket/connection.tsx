import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { getAccessToken } from "../api/client";
import type { AgentPresence, DashboardNumbers, DemoRun, LiveEvent, WaitingTicket } from "../api/types";
import { useAuth } from "../auth/AuthProvider";

type LiveValue = {
  queue: WaitingTicket[] | null;
  agents: AgentPresence[];
  dashboard: DashboardNumbers | null;
  demo: DemoRun | null;
  refreshKey: number;
  setQueue: (tickets: WaitingTicket[]) => void;
  setDashboard: (numbers: DashboardNumbers) => void;
  setDemo: (run: DemoRun | null) => void;
  markBusy: (userId: string) => void;
  markAvailability: (userId: string, status: string, name?: string) => void;
};

const LiveContext = createContext<LiveValue | null>(null);

const queueEvents = new Set([
  "ticket_created",
  "ticket_claimed",
  "severity_changed",
  "ticket_closed",
  "ticket_returned",
  "sla_overdue",
]);

export function LiveProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const [queue, setQueue] = useState<WaitingTicket[] | null>(null);
  const [agents, setAgents] = useState<AgentPresence[]>([]);
  const [dashboard, setDashboard] = useState<DashboardNumbers | null>(null);
  const [demo, setDemo] = useState<DemoRun | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    setQueue(null);
    setAgents([]);
    setDashboard(null);
    setDemo(null);
  }, [user?.id]);

  useEffect(() => {
    if (!user) return;
    let stopped = false;
    let socket: WebSocket | null = null;
    let timer = 0;

    const connect = () => {
      const token = getAccessToken();
      if (!token || stopped) return;
      const protocol = location.protocol === "https:" ? "wss" : "ws";
      socket = new WebSocket(`${protocol}://${location.host}/ws?token=${encodeURIComponent(token)}`);
      socket.onmessage = (event) => {
        let body: LiveEvent;
        try {
          body = JSON.parse(String(event.data)) as LiveEvent;
        } catch {
          return;
        }
        applyEvent(body);
      };
      socket.onclose = () => {
        if (stopped) return;
        timer = window.setTimeout(connect, 2000);
      };
    };

    const applyEvent = (body: LiveEvent) => {
      if (body.type === "queue_snapshot") {
        setQueue(body.tickets ?? []);
        setAgents(body.agents ?? []);
        setRefreshKey((value) => value + 1);
        return;
      }
      if (body.type === "dashboard_updated" && body.dashboard) {
        setDashboard(body.dashboard);
        return;
      }
      if (body.type === "demo_progress") {
        setDemo({
          id: "",
          total_count: body.total_count ?? 0,
          interval_seconds: body.interval_seconds ?? 0,
          distribution: body.distribution ?? "",
          status: body.status ?? "",
          generated_count: body.generated_count ?? 0,
        });
        return;
      }
      if (body.type === "agent_status_changed" && body.user_id && body.status) {
        setAgents((current) => upsertAgent(current, body.user_id!, body.status!, body.name));
      }
      if (body.type === "ticket_claimed" && body.assignee_id) {
        setAgents((current) => upsertAgent(current, body.assignee_id!, "busy"));
      }
      if (queueEvents.has(body.type) || body.type === "agent_status_changed") {
        setRefreshKey((value) => value + 1);
      }
    };

    connect();
    const clock = window.setInterval(() => setRefreshKey((value) => value + 1), 30000);
    return () => {
      stopped = true;
      window.clearTimeout(timer);
      window.clearInterval(clock);
      socket?.close();
    };
  }, [user?.id]);

  const value = useMemo<LiveValue>(
    () => ({
      queue,
      agents,
      dashboard,
      demo,
      refreshKey,
      setQueue,
      setDashboard,
      setDemo,
      markBusy(userId) {
        setAgents((current) => upsertAgent(current, userId, "busy"));
      },
      markAvailability(userId, status, name) {
        setAgents((current) => upsertAgent(current, userId, status, name));
      },
    }),
    [queue, agents, dashboard, demo, refreshKey],
  );

  return <LiveContext.Provider value={value}>{children}</LiveContext.Provider>;
}

function upsertAgent(current: AgentPresence[], userId: string, status: string, name?: string): AgentPresence[] {
  const found = current.some((agent) => agent.user_id === userId);
  if (!found) return [...current, { user_id: userId, name: name ?? "", status }];
  return current.map((agent) =>
    agent.user_id === userId ? { ...agent, status, name: name || agent.name } : agent,
  );
}

export function useLive(): LiveValue {
  const value = useContext(LiveContext);
  if (!value) throw new Error("画面への知らせを画面の外から使っています");
  return value;
}

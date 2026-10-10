import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError, api } from "../../api/client";
import type { WaitingTicket } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { AgentStatusSwitch } from "../../components/agent/AgentStatusSwitch";
import { ClaimNextButton } from "../../components/agent/ClaimNextButton";
import { WaitingQueue } from "../../components/agent/WaitingQueue";
import { ErrorText, PageFrame } from "../../components/common/PageFrame";
import { paths } from "../../paths";
import { useLive } from "../../websocket/connection";

export function QueuePage() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const live = useLive();
  const [tickets, setTickets] = useState<WaitingTicket[] | null>(live.queue);
  const [error, setError] = useState("");
  const mine = live.agents.find((agent) => agent.user_id === user?.id);
  const status = mine?.status ?? "";

  useEffect(() => {
    if (live.queue) setTickets(live.queue);
  }, [live.queue]);

  useEffect(() => {
    api<WaitingTicket[]>("/api/queue")
      .then(setTickets)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "待ち順を表示できませんでした。しばらくしてから、もう一度試してください"));
  }, [live.refreshKey]);

  return (
    <PageFrame title="待ち順">
      <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <AgentStatusSwitch
          userId=""
          status={status}
          onChanged={(next) => {
            if (user) live.markAvailability(user.id, next, user.name);
          }}
        />
        <ClaimNextButton
          enabled={status === "available"}
          onClaimed={(ticket) => {
            if (user) live.markBusy(user.id);
            navigate(paths.agentTicket(ticket.id));
          }}
        />
      </div>
      <ErrorText message={error} />
      {tickets === null && !error && <p className="text-sm text-stone-600">読み込んでいます。</p>}
      {tickets && <WaitingQueue tickets={tickets} hrefFor={paths.agentTicket} />}
    </PageFrame>
  );
}

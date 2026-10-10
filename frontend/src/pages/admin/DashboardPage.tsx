import { useEffect, useState } from "react";
import { ApiError, api } from "../../api/client";
import type { DashboardNumbers, WaitingTicket } from "../../api/types";
import { DashboardSummary } from "../../components/admin/DashboardSummary";
import { WaitingQueue } from "../../components/agent/WaitingQueue";
import { ErrorText, PageFrame } from "../../components/common/PageFrame";
import { availabilityLabel } from "../../labels";
import { paths } from "../../paths";
import { useLive } from "../../websocket/connection";

export function DashboardPage() {
  const live = useLive();
  const [numbers, setNumbers] = useState<DashboardNumbers | null>(live.dashboard);
  const [tickets, setTickets] = useState<WaitingTicket[] | null>(live.queue);
  const [error, setError] = useState("");

  useEffect(() => {
    if (live.dashboard) setNumbers(live.dashboard);
  }, [live.dashboard]);

  useEffect(() => {
    if (live.queue) setTickets(live.queue);
  }, [live.queue]);

  useEffect(() => {
    api<DashboardNumbers>("/api/admin/dashboard")
      .then(setNumbers)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "現場の数字をまとめられませんでした。しばらくしてから、もう一度試してください"));
    api<WaitingTicket[]>("/api/queue")
      .then(setTickets)
      .catch((err: unknown) => {
        if (!error) setError(err instanceof ApiError ? err.message : "待ち順を表示できませんでした。しばらくしてから、もう一度試してください");
      });
  }, [live.refreshKey]);

  return (
    <PageFrame title="現場の数字">
      <ErrorText message={error} />
      {numbers === null && !error && <p className="text-sm text-stone-600">読み込んでいます。</p>}
      {numbers && <DashboardSummary numbers={numbers} />}
      <section className="mt-8">
        <h2 className="mb-3 font-medium">担当者の稼働</h2>
        {live.agents.length === 0 && <p className="text-sm text-stone-600">つながると、担当者の稼働が出ます。</p>}
        {live.agents.length > 0 && (
          <ul className="flex flex-wrap gap-2">
            {live.agents.map((agent) => (
              <li key={agent.user_id} className="rounded-md border border-stone-200 bg-white px-3 py-2 text-sm">
                {agent.name} · {availabilityLabel(agent.status)}
              </li>
            ))}
          </ul>
        )}
      </section>
      <section className="mt-8">
        <h2 className="mb-3 font-medium">待ち順</h2>
        {tickets === null && <p className="text-sm text-stone-600">読み込んでいます。</p>}
        {tickets && <WaitingQueue tickets={tickets} hrefFor={paths.adminTicket} />}
      </section>
    </PageFrame>
  );
}

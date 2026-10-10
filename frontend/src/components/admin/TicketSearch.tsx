import { useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { ApiError, api } from "../../api/client";
import type { AdminTicket, Customer } from "../../api/types";
import { deadlineText, formatWhen, planLabel, severityLabel, ticketStatusLabel } from "../../labels";
import { paths } from "../../paths";
import { buttonClass, ErrorText, fieldClass, quietButtonClass } from "../common/PageFrame";

export function TicketSearch({ customers }: { customers: Customer[] }) {
  const [status, setStatus] = useState("");
  const [severity, setSeverity] = useState("");
  const [customerId, setCustomerId] = useState("");
  const [assigneeId, setAssigneeId] = useState("");
  const [tickets, setTickets] = useState<AdminTicket[] | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  const search = (event?: FormEvent) => {
    event?.preventDefault();
    void load(status, severity, customerId, assigneeId);
  };

  const load = (nextStatus: string, nextSeverity: string, nextCustomerId: string, nextAssigneeId: string) => {
    const params = new URLSearchParams();
    if (nextStatus) params.set("status", nextStatus);
    if (nextSeverity) params.set("severity", nextSeverity);
    if (nextCustomerId) params.set("customer_id", nextCustomerId);
    if (nextAssigneeId.trim()) params.set("assignee_id", nextAssigneeId.trim());
    const query = params.toString();
    setPending(true);
    setError("");
    api<AdminTicket[]>(`/api/admin/tickets${query ? `?${query}` : ""}`)
      .then(setTickets)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "チケットの一覧をまとめられませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  useEffect(() => {
    load("", "", "", "");
  }, []);

  return (
    <div className="space-y-4">
      <form className="grid gap-3 rounded-lg border border-stone-200 bg-white p-4 md:grid-cols-2 xl:grid-cols-5" onSubmit={search}>
        <label className="block text-sm">
          状態
          <select className={fieldClass} value={status} onChange={(event) => setStatus(event.target.value)}>
            <option value="">すべて</option>
            <option value="open">対応待ち</option>
            <option value="in_progress">対応中</option>
            <option value="closed">完了</option>
          </select>
        </label>
        <label className="block text-sm">
          緊急度
          <select className={fieldClass} value={severity} onChange={(event) => setSeverity(event.target.value)}>
            <option value="">すべて</option>
            <option value="1">1 質問・要望</option>
            <option value="2">2 不具合報告</option>
            <option value="3">3 一部機能不可</option>
            <option value="4">4 システム全停止</option>
          </select>
        </label>
        <label className="block text-sm">
          顧客
          <select className={fieldClass} value={customerId} onChange={(event) => setCustomerId(event.target.value)}>
            <option value="">すべて</option>
            {customers.map((customer) => (
              <option key={customer.id} value={customer.id}>
                {customer.name}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          担当者ID
          <input className={fieldClass} value={assigneeId} onChange={(event) => setAssigneeId(event.target.value)} placeholder="空ならすべて" />
        </label>
        <div className="flex items-end gap-2">
          <button className={buttonClass} type="submit" disabled={pending}>
            {pending ? "探しています" : "絞り込む"}
          </button>
          <button
            className={quietButtonClass}
            type="button"
            onClick={() => {
              setStatus("");
              setSeverity("");
              setCustomerId("");
              setAssigneeId("");
            }}
          >
            条件を消す
          </button>
        </div>
      </form>
      <ErrorText message={error} />
      {tickets && tickets.length === 0 && <p className="text-sm text-stone-600">条件に合うチケットはありません。</p>}
      {tickets && tickets.length > 0 && (
        <div className="overflow-x-auto rounded-lg border border-stone-200 bg-white">
          <table className="min-w-full text-left text-sm">
            <thead className="bg-stone-100 text-stone-600">
              <tr>
                <th className="px-3 py-2 font-medium">件名</th>
                <th className="px-3 py-2 font-medium">緊急度</th>
                <th className="px-3 py-2 font-medium">状態</th>
                <th className="px-3 py-2 font-medium">顧客</th>
                <th className="px-3 py-2 font-medium">プラン</th>
                <th className="px-3 py-2 font-medium">担当</th>
                <th className="px-3 py-2 font-medium">点数</th>
                <th className="px-3 py-2 font-medium">約束時間</th>
                <th className="px-3 py-2 font-medium">作成</th>
              </tr>
            </thead>
            <tbody>
              {tickets.map((ticket) => (
                <tr key={ticket.id} className={ticket.overdue ? "bg-red-50 text-red-900" : "border-t border-stone-100"}>
                  <td className="px-3 py-2">
                    <Link className="underline decoration-stone-400 underline-offset-2" to={paths.adminTicket(ticket.id)}>
                      {ticket.title}
                    </Link>
                  </td>
                  <td className="px-3 py-2">{severityLabel(ticket.severity)}</td>
                  <td className="px-3 py-2">{ticketStatusLabel(ticket.status)}</td>
                  <td className="px-3 py-2">{ticket.customer_name || "—"}</td>
                  <td className="px-3 py-2">{planLabel(ticket.plan)}</td>
                  <td className="px-3 py-2">{ticket.assignee_name || "未割り当て"}</td>
                  <td className="px-3 py-2">{ticket.priority_score}</td>
                  <td className="px-3 py-2">{deadlineText(ticket.remaining_minutes, ticket.overdue, ticket.overdue_minutes)}</td>
                  <td className="px-3 py-2">{formatWhen(ticket.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

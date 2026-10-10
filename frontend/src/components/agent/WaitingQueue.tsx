import { Link } from "react-router-dom";
import type { WaitingTicket } from "../../api/types";
import { deadlineText, planLabel, severityLabel } from "../../labels";

export function WaitingQueue({
  tickets,
  hrefFor,
}: {
  tickets: WaitingTicket[];
  hrefFor: (id: string) => string;
}) {
  if (tickets.length === 0) {
    return <p className="text-sm text-stone-600">待ちチケットはありません。</p>;
  }
  return (
    <div className="overflow-x-auto rounded-lg border border-stone-200 bg-white">
      <table className="min-w-full text-left text-sm">
        <thead className="bg-stone-100 text-stone-600">
          <tr>
            <th className="px-3 py-2 font-medium">順位</th>
            <th className="px-3 py-2 font-medium">件名</th>
            <th className="px-3 py-2 font-medium">緊急度</th>
            <th className="px-3 py-2 font-medium">プラン</th>
            <th className="px-3 py-2 font-medium">顧客</th>
            <th className="px-3 py-2 font-medium">待ち</th>
            <th className="px-3 py-2 font-medium">点数</th>
            <th className="px-3 py-2 font-medium">約束時間</th>
          </tr>
        </thead>
        <tbody>
          {tickets.map((ticket) => (
            <tr key={ticket.id} className={ticket.overdue ? "bg-red-50 text-red-900" : "border-t border-stone-100"}>
              <td className="px-3 py-2">{ticket.rank}</td>
              <td className="px-3 py-2">
                <Link className="underline decoration-stone-400 underline-offset-2" to={hrefFor(ticket.id)}>
                  {ticket.title}
                </Link>
              </td>
              <td className="px-3 py-2">{severityLabel(ticket.severity)}</td>
              <td className="px-3 py-2">{planLabel(ticket.plan)}</td>
              <td className="px-3 py-2">{ticket.customer_name}</td>
              <td className="px-3 py-2">{ticket.wait_minutes}分</td>
              <td className="px-3 py-2">{ticket.priority_score}</td>
              <td className="px-3 py-2">{deadlineText(ticket.remaining_minutes, ticket.overdue, ticket.overdue_minutes)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

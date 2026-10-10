import { Link } from "react-router-dom";
import type { MyTicket } from "../../api/types";
import { deadlineText, formatWhen, severityLabel, ticketStatusLabel } from "../../labels";
import { paths } from "../../paths";

export function MyTicketList({ tickets }: { tickets: MyTicket[] }) {
  if (tickets.length === 0) {
    return <p className="text-sm text-stone-600">まだチケットはありません。</p>;
  }
  return (
    <div className="overflow-x-auto rounded-lg border border-stone-200 bg-white">
      <table className="min-w-full text-left text-sm">
        <thead className="bg-stone-100 text-stone-600">
          <tr>
            <th className="px-3 py-2 font-medium">件名</th>
            <th className="px-3 py-2 font-medium">緊急度</th>
            <th className="px-3 py-2 font-medium">状態</th>
            <th className="px-3 py-2 font-medium">作成</th>
            <th className="px-3 py-2 font-medium">約束時間</th>
            <th className="px-3 py-2 font-medium">担当</th>
          </tr>
        </thead>
        <tbody>
          {tickets.map((ticket) => (
            <tr key={ticket.id} className={ticket.overdue ? "bg-red-50 text-red-900" : "border-t border-stone-100"}>
              <td className="px-3 py-2">
                <Link className="underline decoration-stone-400 underline-offset-2" to={paths.ticket(ticket.id)}>
                  {ticket.title}
                </Link>
              </td>
              <td className="px-3 py-2">{severityLabel(ticket.severity)}</td>
              <td className="px-3 py-2">{ticketStatusLabel(ticket.status)}</td>
              <td className="px-3 py-2">{formatWhen(ticket.created_at)}</td>
              <td className="px-3 py-2">{deadlineText(ticket.remaining_minutes, ticket.overdue, ticket.overdue_minutes)}</td>
              <td className="px-3 py-2">{ticket.assignee_name || "未割り当て"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

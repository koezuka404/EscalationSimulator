import { useState } from "react";
import { ApiError, api } from "../../api/client";
import type { Customer, User } from "../../api/types";
import { accountStatusLabel, availabilityLabel, roleLabel } from "../../labels";
import { buttonClass, ErrorText, fieldClass, quietButtonClass } from "../common/PageFrame";

export function UserTable({
  users,
  customers,
  availability,
  onLinked,
  onOffline,
}: {
  users: User[];
  customers: Customer[];
  availability: Record<string, string>;
  onLinked: (user: User) => void;
  onOffline: (userId: string) => void;
}) {
  if (users.length === 0) {
    return <p className="text-sm text-stone-600">利用者はまだいません。</p>;
  }
  return (
    <div className="overflow-x-auto rounded-lg border border-stone-200 bg-white">
      <table className="min-w-full text-left text-sm">
        <thead className="bg-stone-100 text-stone-600">
          <tr>
            <th className="px-3 py-2 font-medium">名前</th>
            <th className="px-3 py-2 font-medium">メール</th>
            <th className="px-3 py-2 font-medium">役割</th>
            <th className="px-3 py-2 font-medium">状態</th>
            <th className="px-3 py-2 font-medium">顧客 / 稼働</th>
          </tr>
        </thead>
        <tbody>
          {users.map((user) => (
            <tr key={user.id} className="border-t border-stone-100 align-top">
              <td className="px-3 py-3">{user.name}</td>
              <td className="px-3 py-3">{user.email}</td>
              <td className="px-3 py-3">{roleLabel(user.role)}</td>
              <td className="px-3 py-3">{accountStatusLabel(user.status)}</td>
              <td className="px-3 py-3">
                {user.role === "applicant" && (
                  <LinkCustomer user={user} customers={customers} onLinked={onLinked} />
                )}
                {user.role === "agent" && (
                  <AgentOffline userId={user.id} status={availability[user.id] ?? ""} onOffline={onOffline} />
                )}
                {user.role === "admin" && <span className="text-stone-500">—</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function LinkCustomer({
  user,
  customers,
  onLinked,
}: {
  user: User;
  customers: Customer[];
  onLinked: (user: User) => void;
}) {
  const [customerId, setCustomerId] = useState(user.customer_id);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  return (
    <div className="flex flex-wrap items-center gap-2">
      <select className="rounded-md border border-stone-300 bg-white px-3 py-2 text-sm" value={customerId} onChange={(event) => setCustomerId(event.target.value)}>
        <option value="">顧客を選ぶ</option>
        {customers.map((customer) => (
          <option key={customer.id} value={customer.id}>
            {customer.name}
          </option>
        ))}
      </select>
      <button
        type="button"
        className={quietButtonClass}
        disabled={pending || !customerId || customerId === user.customer_id}
        onClick={() => {
          setPending(true);
          setError("");
          api<User>(`/api/users/${user.id}`, {
            method: "PATCH",
            body: JSON.stringify({ customer_id: customerId }),
          })
            .then(onLinked)
            .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "申請者を顧客に結びつけられませんでした。しばらくしてから、もう一度試してください"))
            .finally(() => setPending(false));
        }}
      >
        結びつける
      </button>
      {!user.customer_id && <span className="text-amber-800">未設定</span>}
      <ErrorText message={error} />
    </div>
  );
}

function AgentOffline({ userId, status, onOffline }: { userId: string; status: string; onOffline: (userId: string) => void }) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span>{status ? availabilityLabel(status) : "稼働は待ち順の接続後に出ます"}</span>
      <button
        type="button"
        className={buttonClass}
        disabled={pending || status === "offline" || !status}
        onClick={() => {
          setPending(true);
          setError("");
          api("/api/agent/status", {
            method: "PATCH",
            body: JSON.stringify({ status: "offline", user_id: userId }),
          })
            .then(() => onOffline(userId))
            .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "稼働を切り替えられませんでした。しばらくしてから、もう一度試してください"))
            .finally(() => setPending(false));
        }}
      >
        離席にする
      </button>
      <ErrorText message={error} />
    </div>
  );
}

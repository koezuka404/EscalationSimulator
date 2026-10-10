import { useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { ApiError, api } from "../../api/client";
import type { TicketDetail, WorkNote } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { categoryLabel, formatWhen, planLabel, severityLabel, ticketStatusLabel } from "../../labels";
import { useLive } from "../../websocket/connection";
import { buttonClass, ErrorText, fieldClass, quietButtonClass } from "../common/PageFrame";

export function TicketView({ id, backTo, backLabel }: { id: string; backTo: string; backLabel: string }) {
  const { user } = useAuth();
  const { refreshKey, markAvailability } = useLive();
  const [ticket, setTicket] = useState<TicketDetail | null>(null);
  const [error, setError] = useState("");

  const load = () => {
    api<TicketDetail>(`/api/tickets/${id}`)
      .then((detail) => {
        setTicket(detail);
        setError("");
      })
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "チケットの詳細を返せませんでした。しばらくしてから、もう一度試してください"));
  };

  useEffect(() => {
    load();
  }, [id, refreshKey]);

  if (error && !ticket) return <ErrorText message={error} />;
  if (!ticket || !user) return <p className="text-sm text-stone-600">読み込んでいます。</p>;

  const mine = ticket.assignee_id === user.id;
  const open = ticket.status === "open" || ticket.status === "in_progress";
  const canNote = (user.role === "agent" && mine && ticket.status === "in_progress") || (user.role === "admin" && open);
  const canClose = (user.role === "agent" && mine && ticket.status === "in_progress") || (user.role === "admin" && ticket.status === "in_progress");
  const canSeverity = (user.role === "agent" && mine && ticket.status === "in_progress") || (user.role === "admin" && open);
  const canRelease = user.role === "admin" && ticket.status === "in_progress";

  return (
    <div className="space-y-6">
      <Link className="text-sm text-stone-600 underline" to={backTo}>
        {backLabel}
      </Link>
      {error && <ErrorText message={error} />}
      <section className="rounded-lg border border-stone-200 bg-white p-4">
        <h2 className="text-xl font-semibold">{ticket.title}</h2>
        <dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
          <Item label="状態" value={ticketStatusLabel(ticket.status)} />
          <Item label="緊急度" value={severityLabel(ticket.severity)} />
          <Item label="種類" value={categoryLabel(ticket.category)} />
          <Item label="顧客" value={ticket.customer_name || "—"} />
          <Item label="プラン" value={planLabel(ticket.plan)} />
          <Item label="担当" value={ticket.assignee_name || "未割り当て"} />
          <Item label="点数" value={String(ticket.priority_score)} />
          <Item label="作成" value={formatWhen(ticket.created_at)} />
          <Item label="引き取り" value={formatWhen(ticket.claimed_at)} />
          <Item label="完了" value={formatWhen(ticket.closed_at)} />
        </dl>
        <p className="mt-4 whitespace-pre-wrap text-sm leading-6">{ticket.description}</p>
        {ticket.close_comment && (
          <p className="mt-4 whitespace-pre-wrap rounded-md bg-stone-50 px-3 py-2 text-sm">終了コメント: {ticket.close_comment}</p>
        )}
      </section>

      {canSeverity && <SeverityForm ticket={ticket} role={user.role} onDone={load} />}
      {canNote && <NoteForm id={ticket.id} onDone={load} />}
      {canClose && (
        <CloseForm
          id={ticket.id}
          onDone={() => {
            if (user.role === "agent") markAvailability(user.id, "available");
            load();
          }}
        />
      )}
      {canRelease && (
        <ReleaseButton
          id={ticket.id}
          onDone={() => {
            if (ticket.assignee_id) markAvailability(ticket.assignee_id, "available");
            load();
          }}
        />
      )}

      <section>
        <h3 className="font-medium">対応メモ</h3>
        {ticket.work_notes.length === 0 && <p className="mt-2 text-sm text-stone-600">対応メモはありません。</p>}
        <ul className="mt-2 space-y-2">
          {ticket.work_notes.map((note) => (
            <li key={note.id} className="rounded-md border border-stone-200 bg-white px-3 py-2 text-sm">
              <p className="text-stone-500">
                {note.author_name} · {formatWhen(note.created_at)}
              </p>
              <p className="mt-1 whitespace-pre-wrap">{note.body}</p>
            </li>
          ))}
        </ul>
      </section>

      <section>
        <h3 className="font-medium">緊急度の履歴</h3>
        {ticket.severity_changes.length === 0 && <p className="mt-2 text-sm text-stone-600">変更はありません。</p>}
        <ul className="mt-2 space-y-2">
          {ticket.severity_changes.map((change, index) => (
            <li key={`${change.created_at}-${index}`} className="rounded-md border border-stone-200 bg-white px-3 py-2 text-sm">
              <p>
                {severityLabel(change.from_severity)} → {severityLabel(change.to_severity)} · {change.changed_by_name} · {formatWhen(change.created_at)}
              </p>
              <p className="mt-1 whitespace-pre-wrap">{change.reason}</p>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}

function Item({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-stone-500">{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function SeverityForm({ ticket, role, onDone }: { ticket: TicketDetail; role: string; onDone: () => void }) {
  const choices = [1, 2, 3, 4].filter((value) => (role === "agent" ? value > ticket.severity : value !== ticket.severity));
  const [severity, setSeverity] = useState(choices[0] ?? ticket.severity);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  if (choices.length === 0) return null;

  const submit = (event: FormEvent) => {
    event.preventDefault();
    setPending(true);
    setError("");
    api(`/api/tickets/${ticket.id}/escalate`, {
      method: "POST",
      body: JSON.stringify({ severity, reason }),
    })
      .then(() => {
        setReason("");
        onDone();
      })
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "緊急度を変更できませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  return (
    <form className="max-w-xl space-y-3 rounded-lg border border-stone-200 bg-white p-4" onSubmit={submit}>
      <h3 className="font-medium">緊急度を変える</h3>
      <label className="block text-sm">
        新しい緊急度
        <select className={fieldClass} value={severity} onChange={(event) => setSeverity(Number(event.target.value))}>
          {choices.map((value) => (
            <option key={value} value={value}>
              {severityLabel(value)}
            </option>
          ))}
        </select>
      </label>
      <label className="block text-sm">
        理由
        <textarea className={fieldClass} rows={3} maxLength={500} value={reason} onChange={(event) => setReason(event.target.value)} required />
      </label>
      <button className={buttonClass} type="submit" disabled={pending}>
        緊急度を変える
      </button>
      <ErrorText message={error} />
    </form>
  );
}

function NoteForm({ id, onDone }: { id: string; onDone: () => void }) {
  const [body, setBody] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const submit = (event: FormEvent) => {
    event.preventDefault();
    setPending(true);
    setError("");
    api<WorkNote>(`/api/tickets/${id}/comments`, {
      method: "POST",
      body: JSON.stringify({ body }),
    })
      .then(() => {
        setBody("");
        onDone();
      })
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "対応メモを保存できませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };
  return (
    <form className="max-w-xl space-y-3 rounded-lg border border-stone-200 bg-white p-4" onSubmit={submit}>
      <h3 className="font-medium">対応メモ</h3>
      <textarea className={fieldClass} rows={4} maxLength={5000} value={body} onChange={(event) => setBody(event.target.value)} required />
      <button className={buttonClass} type="submit" disabled={pending}>
        メモを追加
      </button>
      <ErrorText message={error} />
    </form>
  );
}

function CloseForm({ id, onDone }: { id: string; onDone: () => void }) {
  const [comment, setComment] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const submit = (event: FormEvent) => {
    event.preventDefault();
    setPending(true);
    setError("");
    api(`/api/tickets/${id}/close`, {
      method: "POST",
      body: JSON.stringify({ comment }),
    })
      .then(() => onDone())
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "チケットを完了できませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };
  return (
    <form className="max-w-xl space-y-3 rounded-lg border border-stone-200 bg-white p-4" onSubmit={submit}>
      <h3 className="font-medium">対応を完了する</h3>
      <textarea className={fieldClass} rows={4} maxLength={2000} value={comment} onChange={(event) => setComment(event.target.value)} required />
      <button className={buttonClass} type="submit" disabled={pending}>
        完了する
      </button>
      <ErrorText message={error} />
    </form>
  );
}

function ReleaseButton({ id, onDone }: { id: string; onDone: () => void }) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  return (
    <div>
      <button
        type="button"
        className={quietButtonClass}
        disabled={pending}
        onClick={() => {
          setPending(true);
          setError("");
          api(`/api/tickets/${id}/release`, { method: "POST" })
            .then(() => onDone())
            .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "チケットを待ちに戻せませんでした。しばらくしてから、もう一度試してください"))
            .finally(() => setPending(false));
        }}
      >
        担当を外して待ちに戻す
      </button>
      <ErrorText message={error} />
    </div>
  );
}

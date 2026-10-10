import { useState, type FormEvent } from "react";
import { ApiError, api } from "../../api/client";
import type { CreatedTicket } from "../../api/types";
import { buttonClass, ErrorText, fieldClass } from "../common/PageFrame";

const categories = [
  ["incident", "障害"],
  ["bug", "不具合"],
  ["question", "質問"],
  ["request", "要望"],
  ["other", "その他"],
] as const;

const severities = [
  [1, "1 質問・要望"],
  [2, "2 不具合報告"],
  [3, "3 一部機能不可"],
  [4, "4 システム全停止"],
] as const;

export function NewTicketForm({
  locked,
  onCreated,
}: {
  locked: boolean;
  onCreated: (ticket: CreatedTicket) => void;
}) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState("question");
  const [severity, setSeverity] = useState(1);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  const submit = (event: FormEvent) => {
    event.preventDefault();
    setPending(true);
    setError("");
    api<CreatedTicket>("/api/tickets", {
      method: "POST",
      body: JSON.stringify({ title, description, category, severity }),
    })
      .then(onCreated)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "チケットを起票できませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  return (
    <form className="max-w-xl space-y-4" onSubmit={submit}>
      {locked && (
        <p className="rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-900">
          所属顧客が設定されていません。管理者に顧客への結びつけを依頼してください。
        </p>
      )}
      <label className="block text-sm">
        件名
        <input className={fieldClass} value={title} maxLength={200} onChange={(event) => setTitle(event.target.value)} required />
      </label>
      <label className="block text-sm">
        詳細
        <textarea className={fieldClass} rows={6} value={description} maxLength={5000} onChange={(event) => setDescription(event.target.value)} required />
      </label>
      <label className="block text-sm">
        種類
        <select className={fieldClass} value={category} onChange={(event) => setCategory(event.target.value)}>
          {categories.map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </label>
      <label className="block text-sm">
        緊急度
        <select className={fieldClass} value={severity} onChange={(event) => setSeverity(Number(event.target.value))}>
          {severities.map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </label>
      <button className={buttonClass} type="submit" disabled={locked || pending}>
        {pending ? "起票しています" : "起票する"}
      </button>
      <ErrorText message={error} />
    </form>
  );
}

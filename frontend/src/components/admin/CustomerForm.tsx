import { useState, type FormEvent } from "react";
import { ApiError, api } from "../../api/client";
import type { Customer } from "../../api/types";
import { planLabel } from "../../labels";
import { buttonClass, ErrorText, fieldClass } from "../common/PageFrame";

export function CustomerForm({ customer, onSaved }: { customer: Customer; onSaved: (customer: Customer) => void }) {
  const [name, setName] = useState(customer.name);
  const [plan, setPlan] = useState(customer.plan);
  const [sla, setSla] = useState(String(customer.sla_minutes));
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  const submit = (event: FormEvent) => {
    event.preventDefault();
    setPending(true);
    setError("");
    setSaved(false);
    api<Customer>(`/admin/customers/${customer.id}`, {
      method: "PUT",
      body: JSON.stringify({ name, plan, sla_minutes: Number(sla) }),
    })
      .then((next) => {
        onSaved(next);
        setSaved(true);
      })
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "顧客を保存できませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  return (
    <form className="rounded-lg border border-stone-200 bg-white p-4" onSubmit={submit}>
      <p className="text-sm text-stone-500">{planLabel(customer.plan)}</p>
      <div className="mt-3 grid gap-3 md:grid-cols-[1fr_160px_140px_auto] md:items-end">
        <label className="block text-sm">
          顧客名
          <input className={fieldClass} value={name} maxLength={100} onChange={(event) => setName(event.target.value)} required />
        </label>
        <label className="block text-sm">
          プラン
          <select className={fieldClass} value={plan} onChange={(event) => setPlan(event.target.value)}>
            <option value="free">無償</option>
            <option value="pro">有償</option>
            <option value="enterprise">VIP</option>
          </select>
        </label>
        <label className="block text-sm">
          約束時間（分）
          <input className={fieldClass} type="number" min={1} max={1440} value={sla} onChange={(event) => setSla(event.target.value)} required />
        </label>
        <button className={buttonClass} type="submit" disabled={pending}>
          {pending ? "保存しています" : "保存"}
        </button>
      </div>
      {saved && <p className="mt-2 text-sm text-stone-600">保存しました。</p>}
      <ErrorText message={error} />
    </form>
  );
}

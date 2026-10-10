import { useEffect, useState, type FormEvent } from "react";
import { ApiError, api } from "../../api/client";
import type { Customer, User } from "../../api/types";
import { UserTable } from "../../components/admin/UserTable";
import { buttonClass, ErrorText, fieldClass, PageFrame } from "../../components/common/PageFrame";
import { useLive } from "../../websocket/connection";

export function UsersPage() {
  const live = useLive();
  const [users, setUsers] = useState<User[] | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [error, setError] = useState("");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [created, setCreated] = useState("");

  const load = () => {
    api<User[]>("/api/users")
      .then(setUsers)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "利用者の一覧をまとめられませんでした。しばらくしてから、もう一度試してください"));
  };

  useEffect(() => {
    load();
    api<Customer[]>("/admin/customers")
      .then(setCustomers)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "顧客を保存できませんでした。しばらくしてから、もう一度試してください"));
  }, []);

  const availability: Record<string, string> = {};
  for (const agent of live.agents) availability[agent.user_id] = agent.status;

  const create = (event: FormEvent) => {
    event.preventDefault();
    setPending(true);
    setCreated("");
    setError("");
    api<User>("/api/users", {
      method: "POST",
      body: JSON.stringify({ name, email, password }),
    })
      .then((user) => {
        setName("");
        setEmail("");
        setPassword("");
        setCreated(`${user.name} を担当者として登録しました。最初の稼働は待機中です。`);
        load();
      })
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "担当者を登録できませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  return (
    <PageFrame title="利用者">
      <form className="mb-8 max-w-xl space-y-3 rounded-lg border border-stone-200 bg-white p-4" onSubmit={create}>
        <h2 className="font-medium">担当者を登録</h2>
        <label className="block text-sm">
          名前
          <input className={fieldClass} value={name} maxLength={50} onChange={(event) => setName(event.target.value)} required />
        </label>
        <label className="block text-sm">
          メールアドレス
          <input className={fieldClass} type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
        </label>
        <label className="block text-sm">
          パスワード
          <input className={fieldClass} type="password" value={password} minLength={8} maxLength={15} onChange={(event) => setPassword(event.target.value)} required />
        </label>
        <button className={buttonClass} type="submit" disabled={pending}>
          {pending ? "登録しています" : "担当者を登録"}
        </button>
        {created && <p className="text-sm text-stone-600">{created}</p>}
      </form>
      <ErrorText message={error} />
      {users === null && !error && <p className="text-sm text-stone-600">読み込んでいます。</p>}
      {users && (
        <UserTable
          users={users}
          customers={customers}
          availability={availability}
          onLinked={(user) => setUsers((current) => current?.map((item) => (item.id === user.id ? user : item)) ?? [user])}
          onOffline={(userId) => live.markAvailability(userId, "offline")}
        />
      )}
    </PageFrame>
  );
}

import { useState, type FormEvent } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { ApiError, api } from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import { homeFor, paths } from "../paths";

const inputClass =
  "mt-2 w-full rounded-lg border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none focus:border-blue-500";

export function AuthScreen({ mode }: { mode: "login" | "register" }) {
  const { user, ready, login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const notice = (location.state as { message?: string } | null)?.message ?? "";
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  if (ready && user) return <Navigate to={homeFor(user.role)} replace />;

  const submit = (event: FormEvent) => {
    event.preventDefault();
    setPending(true);
    setError("");
    if (mode === "login") {
      login(email, password)
        .then((person) => navigate(homeFor(person.role)))
        .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "ログインできませんでした。しばらくしてから、もう一度試してください"))
        .finally(() => setPending(false));
      return;
    }
    api("/api/auth/register", {
      method: "POST",
      body: JSON.stringify({ name, email, password }),
    })
      .then(() => login(email, password))
      .then((person) => navigate(homeFor(person.role)))
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "登録できませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  const loginLabel = pending && mode === "login" ? "ログインしています" : "ログイン";
  const registerLabel = pending && mode === "register" ? "登録しています" : "新規登録";

  return (
    <div className="flex min-h-screen items-center justify-center bg-[#f4f6f8] px-4 py-10">
      <div className="w-full max-w-[440px] rounded-2xl bg-white px-8 py-8 shadow-[0_10px_30px_rgba(15,23,42,0.08)]">
        <h1 className="text-3xl font-semibold tracking-tight text-slate-900">エスカレシミュレーター</h1>
        <div className="mt-6 grid grid-cols-2 gap-3">
          <Link
            to={paths.login}
            className={mode === "login" ? "block w-full rounded-lg bg-slate-900 py-2.5 text-center text-sm font-medium text-white" : "block w-full rounded-lg border border-slate-200 bg-white py-2.5 text-center text-sm font-medium text-slate-800"}
          >
            ログイン
          </Link>
          <Link
            to={paths.register}
            className={mode === "register" ? "block w-full rounded-lg bg-slate-900 py-2.5 text-center text-sm font-medium text-white" : "block w-full rounded-lg border border-slate-200 bg-white py-2.5 text-center text-sm font-medium text-slate-800"}
          >
            新規登録
          </Link>
        </div>
        {notice && <p className="mt-4 text-sm text-slate-600">{notice}</p>}
        {!ready && <p className="mt-4 text-sm text-slate-500">ログインを確認しています。</p>}
        <form className="mt-6" onSubmit={submit}>
          {mode === "register" && (
            <label className="block text-sm font-medium text-slate-800">
              名前
              <input className={inputClass} value={name} maxLength={50} onChange={(event) => setName(event.target.value)} required />
            </label>
          )}
          <label className={`block text-sm font-medium text-slate-800 ${mode === "register" ? "mt-5" : ""}`}>
            メールアドレス
            <input className={inputClass} type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
          </label>
          <label className="mt-5 block text-sm font-medium text-slate-800">
            パスワード
            <input
              className={inputClass}
              type="password"
              value={password}
              minLength={mode === "register" ? 8 : undefined}
              maxLength={mode === "register" ? 15 : undefined}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </label>
          {mode === "register" && <p className="mt-2 text-xs text-slate-500">8文字以上15文字以内。英字と数字をそれぞれ1文字以上。</p>}
          <button
            className="mt-6 w-full rounded-lg bg-blue-600 py-3 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
            type="submit"
            disabled={pending || !ready}
          >
            {mode === "login" ? loginLabel : registerLabel}
          </button>
          {error && (
            <p role="alert" className="mt-3 text-sm text-red-700">
              {error}
            </p>
          )}
        </form>
      </div>
    </div>
  );
}

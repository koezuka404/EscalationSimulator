import { useState, type FormEvent } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { ApiError } from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import { buttonClass, ErrorText, fieldClass } from "../components/common/PageFrame";
import { homeFor, paths } from "../paths";

export function LoginPage() {
  const { user, ready, login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const notice = (location.state as { message?: string } | null)?.message ?? "";
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  if (ready && user) return <Navigate to={homeFor(user.role)} replace />;

  const submit = (event: FormEvent) => {
    event.preventDefault();
    setPending(true);
    setError("");
    login(email, password)
      .then((person) => navigate(homeFor(person.role)))
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "ログインできませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-6">
      <h1 className="text-2xl font-semibold">ログイン</h1>
      <p className="mt-2 text-sm text-stone-600">エスカレシミュレーター</p>
      {notice && <p className="mt-4 text-sm text-stone-700">{notice}</p>}
      {!ready && <p className="mt-4 text-sm text-stone-600">ログインを確認しています。</p>}
      <form className="mt-6 space-y-4" onSubmit={submit}>
        <label className="block text-sm">
          メールアドレス
          <input className={fieldClass} type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
        </label>
        <label className="block text-sm">
          パスワード
          <input className={fieldClass} type="password" value={password} onChange={(event) => setPassword(event.target.value)} required />
        </label>
        <button className={buttonClass} type="submit" disabled={pending || !ready}>
          {pending ? "ログインしています" : "ログイン"}
        </button>
        <ErrorText message={error} />
      </form>
      <p className="mt-6 text-sm">
        アカウントが無いときは <Link className="underline" to={paths.register}>会員登録</Link>
      </p>
    </div>
  );
}

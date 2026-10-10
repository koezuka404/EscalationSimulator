import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { ApiError, api } from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import { buttonClass, ErrorText, fieldClass } from "../components/common/PageFrame";
import { homeFor, paths } from "../paths";

export function RegisterPage() {
  const { user, ready } = useAuth();
  const navigate = useNavigate();
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
    api("/api/auth/register", {
      method: "POST",
      body: JSON.stringify({ name, email, password }),
    })
      .then(() => navigate(paths.login, { state: { message: "登録しました。ログインしてください。" } }))
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "登録できませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-6">
      <h1 className="text-2xl font-semibold">会員登録</h1>
      <p className="mt-2 text-sm text-stone-600">申請者として登録します。起票するには、管理者が顧客を結びつけます。</p>
      <form className="mt-6 space-y-4" onSubmit={submit}>
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
          <span className="mt-1 block text-xs text-stone-500">8文字以上15文字以内。英字と数字をそれぞれ1文字以上。</span>
        </label>
        <button className={buttonClass} type="submit" disabled={pending}>
          {pending ? "登録しています" : "登録する"}
        </button>
        <ErrorText message={error} />
      </form>
      <p className="mt-6 text-sm">
        登録済みのときは <Link className="underline" to={paths.login}>ログイン</Link>
      </p>
    </div>
  );
}

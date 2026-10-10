import { useState, type FormEvent } from "react";
import { ApiError, api } from "../../api/client";
import type { DemoRun } from "../../api/types";
import { demoStatusLabel, distributionLabel } from "../../labels";
import { buttonClass, ErrorText, fieldClass, quietButtonClass } from "../common/PageFrame";

export function DemoForm({
  progress,
  onStarted,
  onStopped,
}: {
  progress: DemoRun | null;
  onStarted: (run: DemoRun) => void;
  onStopped: (run: DemoRun) => void;
}) {
  const [count, setCount] = useState(5);
  const [interval, setIntervalSeconds] = useState(2);
  const [distribution, setDistribution] = useState("even");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  const start = (event: FormEvent) => {
    event.preventDefault();
    setPending(true);
    setError("");
    api<DemoRun>("/api/admin/demo/start", {
      method: "POST",
      body: JSON.stringify({ count, interval_seconds: interval, distribution }),
    })
      .then(onStarted)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "デモを始められませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  const stop = () => {
    setPending(true);
    setError("");
    api<DemoRun>("/api/admin/demo/stop", { method: "POST" })
      .then(onStopped)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "デモを止められませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  return (
    <div className="max-w-xl space-y-4">
      <form className="space-y-4 rounded-lg border border-stone-200 bg-white p-4" onSubmit={start}>
        <label className="block text-sm">
          作る件数（1〜50）
          <input className={fieldClass} type="number" min={1} max={50} value={count} onChange={(event) => setCount(Number(event.target.value))} required />
        </label>
        <label className="block text-sm">
          間隔（秒、1〜10）
          <input className={fieldClass} type="number" min={1} max={10} value={interval} onChange={(event) => setIntervalSeconds(Number(event.target.value))} required />
        </label>
        <label className="block text-sm">
          出方
          <select className={fieldClass} value={distribution} onChange={(event) => setDistribution(event.target.value)}>
            <option value="even">均等</option>
            <option value="incident">障害多め</option>
          </select>
        </label>
        <div className="flex gap-2">
          <button className={buttonClass} type="submit" disabled={pending}>
            デモを始める
          </button>
          <button className={quietButtonClass} type="button" disabled={pending} onClick={stop}>
            デモを止める
          </button>
        </div>
        <ErrorText message={error} />
      </form>
      {progress && (
        <p className="text-sm text-stone-700">
          {distributionLabel(progress.distribution)} · {demoStatusLabel(progress.status)} · {progress.generated_count} / {progress.total_count}件
          {progress.interval_seconds ? ` · ${progress.interval_seconds}秒間隔` : ""}
        </p>
      )}
    </div>
  );
}

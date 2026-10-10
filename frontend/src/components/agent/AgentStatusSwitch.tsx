import { useState } from "react";
import { ApiError, api } from "../../api/client";
import { availabilityLabel } from "../../labels";
import { buttonClass, quietButtonClass } from "../common/PageFrame";
import { ErrorText } from "../common/PageFrame";

export function AgentStatusSwitch({
  userId,
  status,
  onChanged,
}: {
  userId: string;
  status: string;
  onChanged: (status: string) => void;
}) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  const change = (next: "available" | "offline") => {
    setPending(true);
    setError("");
    api<{ status: string }>("/api/agent/status", {
      method: "PATCH",
      body: JSON.stringify({ status: next, user_id: userId }),
    })
      .then((body) => onChanged(body.status))
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "稼働を切り替えられませんでした。しばらくしてから、もう一度試してください"))
      .finally(() => setPending(false));
  };

  return (
    <div>
      <p className="text-sm text-stone-700">
        今の稼働: <span className="font-medium">{status ? availabilityLabel(status) : "確認しています"}</span>
      </p>
      <div className="mt-3 flex gap-2">
        <button type="button" className={quietButtonClass} disabled={pending || status === "available" || status === "busy"} onClick={() => change("available")}>
          待機中にする
        </button>
        <button type="button" className={buttonClass} disabled={pending || status === "offline"} onClick={() => change("offline")}>
          離席にする
        </button>
      </div>
      {status === "busy" && <p className="mt-2 text-sm text-stone-600">対応中のチケットがあるため、待機中にはできません。離席にすると、そのチケットは待ちに戻ります。</p>}
      <ErrorText message={error} />
    </div>
  );
}

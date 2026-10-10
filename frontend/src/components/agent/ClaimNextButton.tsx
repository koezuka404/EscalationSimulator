import { useState } from "react";
import { ApiError, api } from "../../api/client";
import type { CreatedTicket } from "../../api/types";
import { buttonClass } from "../common/PageFrame";
import { ErrorText } from "../common/PageFrame";

export function ClaimNextButton({
  enabled,
  onClaimed,
}: {
  enabled: boolean;
  onClaimed: (ticket: CreatedTicket) => void;
}) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  return (
    <div>
      <button
        type="button"
        className={buttonClass}
        disabled={!enabled || pending}
        onClick={() => {
          setPending(true);
          setError("");
          api<CreatedTicket>("/api/queue/claim", { method: "POST" })
            .then(onClaimed)
            .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "チケットを引き取れませんでした。しばらくしてから、もう一度試してください"))
            .finally(() => setPending(false));
        }}
      >
        {pending ? "引き取っています" : "次を引き取る"}
      </button>
      {!enabled && <p className="mt-2 text-sm text-stone-600">待機中のときだけ引き取れます。</p>}
      <ErrorText message={error} />
    </div>
  );
}

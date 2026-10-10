import { useEffect, useState } from "react";
import { ApiError, api } from "../../api/client";
import type { MyTicket } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { MyTicketList } from "../../components/applicant/MyTicketList";
import { ErrorText, PageFrame } from "../../components/common/PageFrame";
import { useLive } from "../../websocket/connection";

export function MyTicketsPage() {
  const { user } = useAuth();
  const { refreshKey } = useLive();
  const [tickets, setTickets] = useState<MyTicket[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<MyTicket[]>("/api/tickets")
      .then(setTickets)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "チケットの一覧を返せませんでした。しばらくしてから、もう一度試してください"));
  }, [refreshKey]);

  return (
    <PageFrame title="自分のチケット">
      {!user?.customer_id && (
        <p className="mb-4 rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-900">
          所属顧客が設定されていません。管理者に顧客への結びつけを依頼してください。
        </p>
      )}
      <ErrorText message={error} />
      {tickets === null && !error && <p className="text-sm text-stone-600">読み込んでいます。</p>}
      {tickets && <MyTicketList tickets={tickets} />}
    </PageFrame>
  );
}

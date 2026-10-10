import { useEffect, useState } from "react";
import { ApiError, api } from "../../api/client";
import type { Customer } from "../../api/types";
import { TicketSearch } from "../../components/admin/TicketSearch";
import { ErrorText, PageFrame } from "../../components/common/PageFrame";

export function AllTicketsPage() {
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    api<Customer[]>("/admin/customers")
      .then(setCustomers)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "顧客を保存できませんでした。しばらくしてから、もう一度試してください"));
  }, []);

  return (
    <PageFrame title="全件">
      <ErrorText message={error} />
      <TicketSearch customers={customers} />
    </PageFrame>
  );
}

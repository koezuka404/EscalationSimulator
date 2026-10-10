import { useEffect, useState } from "react";
import { ApiError, api } from "../../api/client";
import type { Customer } from "../../api/types";
import { CustomerForm } from "../../components/admin/CustomerForm";
import { ErrorText, PageFrame } from "../../components/common/PageFrame";

export function CustomersPage() {
  const [customers, setCustomers] = useState<Customer[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<Customer[]>("/admin/customers")
      .then(setCustomers)
      .catch((err: unknown) => setError(err instanceof ApiError ? err.message : "顧客を保存できませんでした。しばらくしてから、もう一度試してください"));
  }, []);

  return (
    <PageFrame title="顧客">
      <p className="mb-4 text-sm text-stone-600">名前、プラン、初回対応の約束時間を変えられます。変えたあとの新規起票と、対応待ちの点数のやり直しから効きます。</p>
      <ErrorText message={error} />
      {customers === null && !error && <p className="text-sm text-stone-600">読み込んでいます。</p>}
      <div className="space-y-4">
        {customers?.map((customer) => (
          <CustomerForm
            key={customer.id}
            customer={customer}
            onSaved={(next) => setCustomers((current) => current?.map((item) => (item.id === next.id ? next : item)) ?? [next])}
          />
        ))}
      </div>
    </PageFrame>
  );
}

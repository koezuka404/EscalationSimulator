import type { DashboardNumbers } from "../../api/types";

const cards: { key: keyof DashboardNumbers; label: string; hint: string }[] = [
  { key: "waiting_count", label: "待ち件数", hint: "対応待ち" },
  { key: "in_progress_count", label: "対応中件数", hint: "今対応している件数" },
  { key: "average_handle_minutes", label: "平均の対応時間", hint: "直近24時間、分" },
  { key: "average_first_response_minutes", label: "平均の初回応答", hint: "直近24時間、分" },
  { key: "overdue_count", label: "今の約束時間オーバー", hint: "対応待ちで超過" },
  { key: "overdue_24h_count", label: "24時間の約束時間オーバー", hint: "担当が付いた時点で超過" },
  { key: "available_count", label: "待機中", hint: "担当者" },
  { key: "busy_count", label: "作業中", hint: "担当者" },
  { key: "offline_count", label: "離席", hint: "担当者" },
];

export function DashboardSummary({ numbers }: { numbers: DashboardNumbers }) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {cards.map((card) => (
        <article key={card.key} className="rounded-lg border border-stone-200 bg-white px-4 py-3">
          <p className="text-sm text-stone-600">{card.label}</p>
          <p className={`mt-1 text-3xl font-semibold ${card.key === "overdue_count" && numbers.overdue_count > 0 ? "text-red-700" : ""}`}>
            {numbers[card.key]}
          </p>
          <p className="mt-1 text-xs text-stone-500">{card.hint}</p>
        </article>
      ))}
    </div>
  );
}

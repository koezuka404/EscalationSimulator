export function roleLabel(role: string): string {
  if (role === "applicant") return "申請者";
  if (role === "agent") return "担当者";
  if (role === "admin") return "管理者";
  return role;
}

export function accountStatusLabel(status: string): string {
  if (status === "active") return "利用中";
  if (status === "suspended") return "停止";
  if (status === "deleted") return "削除";
  return status;
}

export function ticketStatusLabel(status: string): string {
  if (status === "open") return "対応待ち";
  if (status === "in_progress") return "対応中";
  if (status === "closed") return "完了";
  return status;
}

export function planLabel(plan: string): string {
  if (plan === "free") return "無償";
  if (plan === "pro") return "有償";
  if (plan === "enterprise") return "VIP";
  return plan;
}

export function categoryLabel(category: string): string {
  if (category === "incident") return "障害";
  if (category === "bug") return "不具合";
  if (category === "question") return "質問";
  if (category === "request") return "要望";
  if (category === "other") return "その他";
  return category;
}

export function severityLabel(severity: number): string {
  if (severity === 4) return "4 システム全停止";
  if (severity === 3) return "3 一部機能不可";
  if (severity === 2) return "2 不具合報告";
  if (severity === 1) return "1 質問・要望";
  return String(severity);
}

export function availabilityLabel(status: string): string {
  if (status === "available") return "待機中";
  if (status === "busy") return "作業中";
  if (status === "offline") return "離席";
  return status;
}

export function demoStatusLabel(status: string): string {
  if (status === "running") return "実行中";
  if (status === "completed") return "完了";
  if (status === "stopped") return "停止";
  return status;
}

export function distributionLabel(distribution: string): string {
  if (distribution === "even") return "均等";
  if (distribution === "incident") return "障害多め";
  return distribution;
}

export function formatWhen(value: string | null | undefined): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("ja-JP", {
    month: "numeric",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function deadlineText(remaining: number, overdue: boolean, overdueMinutes: number): string {
  if (overdue) return `${overdueMinutes}分超過`;
  return `残り${remaining}分`;
}

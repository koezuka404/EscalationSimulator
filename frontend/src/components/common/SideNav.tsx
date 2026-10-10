import { NavLink, useNavigate } from "react-router-dom";
import { paths } from "../../paths";
import { roleLabel } from "../../labels";
import { useAuth } from "../../auth/AuthProvider";

const linkClass = ({ isActive }: { isActive: boolean }) =>
  `block rounded-md px-3 py-2 text-sm whitespace-nowrap ${isActive ? "bg-white/15 text-white" : "text-stone-200 hover:bg-white/10"}`;

export function SideNav() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  if (!user) return null;
  const items = navItems(user.role);

  return (
    <aside className="bg-stone-900 text-stone-100 md:min-h-screen">
      <div className="flex items-center justify-between gap-3 px-4 py-4 md:block">
        <div>
          <p className="text-sm font-semibold tracking-wide">エスカレシミュレーター</p>
          <p className="mt-1 text-sm text-stone-300">
            {user.name} · {roleLabel(user.role)}
          </p>
        </div>
        <button
          type="button"
          className="rounded-md px-2 py-1 text-sm text-stone-300 hover:bg-white/10 hover:text-white"
          onClick={() => {
            void logout().then(() => navigate(paths.login));
          }}
        >
          ログアウト
        </button>
      </div>
      <nav className="flex gap-1 overflow-x-auto px-3 pb-4 md:block md:space-y-1 md:px-3">
        {items.map((item) => (
          <NavLink key={item.to} to={item.to} className={linkClass}>
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}

function navItems(role: string): { to: string; label: string }[] {
  if (role === "admin") {
    return [
      { to: paths.dashboard, label: "現場の数字" },
      { to: paths.adminTickets, label: "全件" },
      { to: paths.customers, label: "顧客" },
      { to: paths.users, label: "利用者" },
      { to: paths.demo, label: "デモ" },
    ];
  }
  if (role === "agent") {
    return [{ to: paths.queue, label: "待ち順" }];
  }
  return [
    { to: paths.tickets, label: "自分のチケット" },
    { to: paths.newTicket, label: "起票" },
  ];
}

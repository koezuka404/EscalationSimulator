import type { ReactNode } from "react";
import { SideNav } from "./SideNav";

export function PageFrame({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="min-h-screen md:grid md:grid-cols-[240px_1fr]">
      <SideNav />
      <main className="px-5 py-6 md:px-8 md:py-8">
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        <div className="mt-6">{children}</div>
      </main>
    </div>
  );
}

export function ErrorText({ message }: { message: string }) {
  if (!message) return null;
  return (
    <p role="alert" className="mt-3 text-sm text-red-700">
      {message}
    </p>
  );
}

export const fieldClass =
  "mt-1 w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm outline-none focus:border-stone-800";

export const buttonClass =
  "rounded-md bg-stone-900 px-4 py-2 text-sm font-medium text-white hover:bg-stone-700 disabled:cursor-not-allowed disabled:opacity-50";

export const quietButtonClass =
  "rounded-md border border-stone-300 bg-white px-4 py-2 text-sm hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-50";

"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

export function DashboardNav({ isAdmin, billing }: { isAdmin: boolean; billing: boolean }) {
  const path = usePathname();
  const inside = (href: string) => (path === href || path.startsWith(`${href}/`) ? "page" : undefined);
  return (
    <nav className="inline" aria-label="Account" style={{ gap: 24 }}>
      <Link className="nav-word" href="/dashboard" aria-current={path === "/dashboard" || path.startsWith("/dashboard/play/") ? "page" : undefined}>Your games</Link>
      {billing && <Link className="nav-word" href="/dashboard/billing" aria-current={inside("/dashboard/billing")}>Billing</Link>}
      {isAdmin && <Link className="nav-word" href="/dashboard/admin/users" aria-current={inside("/dashboard/admin")}>Admin</Link>}
    </nav>
  );
}

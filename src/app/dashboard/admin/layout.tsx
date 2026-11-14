import type { Metadata } from "next";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { ApiError, api, type Me } from "@/lib/api";

export const metadata: Metadata = { robots: { index: false } };

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  let me: Me;
  try {
    me = await api.me();
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
    throw e;
  }
  if (!me.user.is_admin) notFound();
  return (
    <>
      <nav className="inline small" aria-label="Admin">
        <Link href="/dashboard/admin/users">Users</Link>
        <Link href="/dashboard/admin/feedback">Feedback</Link>
      </nav>
      {children}
    </>
  );
}

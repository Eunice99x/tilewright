import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { logoutAction } from "@/app/auth/actions";
import { Button } from "@/components/ui/button";
import { ApiError, api, type Me } from "@/lib/api";
import { PLANS } from "@/lib/plans";
import { DashboardNav } from "./nav";

export const metadata: Metadata = { robots: { index: false } };

export default async function DashboardLayout({ children }: { children: React.ReactNode }) {
  let me: Me;
  try {
    me = await api.me();
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
    throw e;
  }
  return (
    <div className="wrap app">
      <div className="app-bar">
        {me.user.is_admin || (!me.user.guest && PLANS.some((p) => p.price > 0))
          ? <DashboardNav isAdmin={me.user.is_admin} billing={!me.user.is_admin && !me.user.guest && PLANS.some((p) => p.price > 0)} />
          : <span />}
        <div className="who">
          <span>{me.user.guest ? `Playing as ${me.user.name}` : me.user.email}</span>
          {me.user.guest && <span title="Guests keep their games in this browser only">Guest: logging out loses your games</span>}
          <form action={logoutAction}><Button variant="ghost">Log out</Button></form>
        </div>
      </div>
      {children}
    </div>
  );
}

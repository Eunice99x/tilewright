import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ApiError, api, type Me } from "@/lib/api";
import { APP, PLANS, priceNote } from "@/lib/plans";
import { portalAction } from "../actions";
import { PaddleButton } from "./paddle-button";

export const metadata: Metadata = { title: "Billing", robots: { index: false } };

const priceId = (plan: string) => process.env[`PADDLE_PRICE_${plan.toUpperCase()}`];

export default async function Billing({ searchParams }: PageProps<"/dashboard/billing">) {
  let me: Me;
  try {
    me = await api.me();
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
    throw e;
  }
  if (me.user.is_admin) redirect("/dashboard/admin");
  const { done, portal } = await searchParams;
  const token = process.env.PADDLE_CLIENT_TOKEN;
  const paddle = !!token && PLANS.every((p) => p.id === "free" || priceId(p.id));
  const current = me.user.plan;
  const subscribed = PLANS.find((p) => p.id === current)?.billing === "subscription";

  return (
    <>
      <div className="app-head"><h1>Billing</h1></div>
      {done && <p className="notice" role="status">Thanks! Your plan updates shortly after payment.</p>}
      {!paddle && <p className="notice">Paid plans are not open yet.</p>}
      {portal && <p className="err" role="alert">Couldn&rsquo;t reach Paddle. Try again, or email {APP.support_email}.</p>}
      {!me.user.email_verified && (
        <p className="notice">Confirm your email to choose a plan. The <a href="/dashboard">dashboard</a> can send the link again.</p>
      )}
      {subscribed && (
        <Card title="Your subscription">
          <p>Cancelling stops the next renewal. You keep your plan until the end of the period you paid for.</p>
          <form action={portalAction} className="inline">
            <Button name="to" value="update_payment">Update card</Button>
            <Button variant="ghost" name="to" value="overview">Invoices</Button>
            <Button variant="danger" name="to" value="cancel">Cancel subscription</Button>
          </form>
        </Card>
      )}
      <div className="plans">
        {PLANS.map((p) => {
          const price = priceId(p.id);
          return (
            <Card key={p.id} title={p.name} className={p.id === current ? "on" : undefined}>
              <p className="big">${p.price}<span className="muted small"> {priceNote(p)}</span></p>
              <ul className="features">{p.features.map((f) => <li key={f}>{f}</li>)}</ul>
              {p.id === current ? (
                <span className="muted">{p.billing === "one_time" ? "Current plan, yours for life" : "Current plan"}</span>
              ) : me.user.email_verified && current === "free" && p.id !== "free" && paddle && token && price ? (
                <PaddleButton token={token} priceId={price} userId={me.user.id} email={me.user.email} label={`Choose ${p.name}`} />
              ) : null}
            </Card>
          );
        })}
      </div>
    </>
  );
}

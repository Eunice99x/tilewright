import type { Metadata } from "next";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { Card } from "@/components/ui/card";
import { ApiError, api, type AdminUserDetail } from "@/lib/api";
import { planName } from "@/lib/plans";
import { fullDay, signIn } from "../../format";

export const metadata: Metadata = { title: "User" };

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export default async function UserPage({ params }: PageProps<"/dashboard/admin/users/[id]">) {
  const { id } = await params;
  if (!UUID.test(id)) notFound();
  let u: AdminUserDetail;
  try {
    u = await api.adminUser(id);
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
    if (e instanceof ApiError && e.status === 404) notFound();
    throw e;
  }
  const none = <span className="muted">None</span>;
  const details: [string, React.ReactNode][] = [
    ["Plan", planName(u.plan)],
    ["Subscription status", u.status || none],
    ["Confirmed", u.confirmed ? "Yes" : "No"],
    ["Sign-in", signIn(u.sign_in)],
    ["Joined", fullDay(u.created_at)],
    ["Lifetime purchase", u.lifetime_plan ? <code key="l">{planName(u.lifetime_plan)} ({u.lifetime_transaction_id})</code> : none],
    ["Paddle customer", u.paddle_customer_id ? <code key="c">{u.paddle_customer_id}</code> : none],
    ["Paddle subscription", u.subscription_id ? <code key="s">{u.subscription_id}</code> : none],
  ];
  return (
    <>
      <div>
        <p className="muted small"><Link href="/dashboard/admin/users">All users</Link></p>
        <h1 className="wrap">{u.email}</h1>
      </div>
      <Card title="Account">
        <dl className="dl">
          {details.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}
        </dl>
      </Card>
    </>
  );
}

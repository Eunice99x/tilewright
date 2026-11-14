import type { Metadata } from "next";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table } from "@/components/ui/table";
import { ApiError, api, type AdminUsersPage } from "@/lib/api";
import { PLANS, planName } from "@/lib/plans";
import { dayTime, fullDay, n, signIn } from "../format";

export const metadata: Metadata = { title: "Users" };

const PATH = "/dashboard/admin/users";
const PLAN_IDS = [...PLANS.map((p) => p.id as string), "admin"];
const one = (v: string | string[] | undefined) => (typeof v === "string" ? v : "");

export default async function Users({ searchParams }: PageProps<"/dashboard/admin/users">) {
  const sp = await searchParams;
  const q = one(sp.q).trim().slice(0, 254);
  const plan = PLAN_IDS.includes(one(sp.plan)) ? one(sp.plan) : "";
  const confirmed = ["true", "false"].includes(one(sp.confirmed)) ? one(sp.confirmed) : "";
  const asked = /^\d{1,15}$/.test(one(sp.offset)) ? Number(one(sp.offset)) : 0;

  let page: AdminUsersPage;
  try {
    page = await api.adminUsers({ q, plan, confirmed, offset: asked });
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
    if (e instanceof ApiError && e.status === 404) notFound();
    throw e;
  }
  const { users, total, offset, page_size: size } = page;
  const filtered = !!(q || plan || confirmed);
  const href = (to: number) => {
    const s = new URLSearchParams();
    if (q) s.set("q", q);
    if (plan) s.set("plan", plan);
    if (confirmed) s.set("confirmed", confirmed);
    if (to > 0) s.set("offset", String(to));
    const qs = s.toString();
    return qs ? `${PATH}?${qs}` : PATH;
  };

  return (
    <>
      <div className="app-head">
        <h1>Users</h1>
        <p className="muted small">{n(total)} {filtered ? (total === 1 ? "match" : "matches") : total === 1 ? "account" : "accounts"}, newest first</p>
      </div>

      <form key={`${q}|${plan}|${confirmed}`} className="inline" action={PATH} role="search">
        <label className="sr" htmlFor="users-q">Search by email</label>
        <Input id="users-q" name="q" type="search" defaultValue={q} placeholder="Search email" maxLength={254} autoCapitalize="none" autoCorrect="off" spellCheck={false} />
        <label className="sr" htmlFor="users-plan">Plan</label>
        <select className="input" id="users-plan" name="plan" defaultValue={plan}>
          <option value="">All plans</option>
          {PLAN_IDS.map((id) => <option key={id} value={id}>{planName(id)}</option>)}
        </select>
        <label className="sr" htmlFor="users-confirmed">Confirmed</label>
        <select className="input" id="users-confirmed" name="confirmed" defaultValue={confirmed}>
          <option value="">Confirmed or not</option>
          <option value="true">Confirmed</option>
          <option value="false">Not confirmed</option>
        </select>
        <Button variant="ghost">Search</Button>
        {filtered && <Link href={PATH}>Clear</Link>}
      </form>

      <Card aria-label="Accounts">
        {users.length > 0 ? (
          <Table>
            <thead>
              <tr><th>Email</th><th>Plan</th><th>Subscription</th><th>Confirmed</th><th>Sign-in</th><th>Joined</th><th>Last sign-in</th></tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td><Link href={`${PATH}/${u.id}`}>{u.email}</Link></td>
                  <td><Badge>{planName(u.plan)}</Badge></td>
                  <td>{u.status || <span className="muted">None</span>}</td>
                  <td>{u.confirmed ? <Badge tone="good">Yes</Badge> : <Badge>No</Badge>}</td>
                  <td>{signIn(u.sign_in)}</td>
                  <td>{fullDay(u.created_at)}</td>
                  <td>{u.last_seen ? dayTime(u.last_seen) : <span className="muted">None</span>}</td>
                </tr>
              ))}
            </tbody>
          </Table>
        ) : total > 0 ? (
          <p className="muted">Nothing on this page. <Link href={href(0)}>Go to the first page</Link></p>
        ) : filtered ? (
          <p className="muted">No account matches. <Link href={PATH}>Clear the search</Link></p>
        ) : (
          <p className="muted">No accounts yet.</p>
        )}
        {users.length > 0 && total > size && (
          <div className="row">
            <span className="muted small">{n(offset + 1)} to {n(offset + users.length)} of {n(total)}</span>
            <span className="inline">
              {offset > 0 && <Link href={href(Math.max(0, offset - size))} rel="prev">Newer</Link>}
              {offset + users.length < total && <Link href={href(offset + size)} rel="next">Older</Link>}
            </span>
          </div>
        )}
      </Card>
    </>
  );
}

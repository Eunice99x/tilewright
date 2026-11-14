import type { Metadata } from "next";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { Card } from "@/components/ui/card";
import { Table } from "@/components/ui/table";
import { ApiError, api, type AdminFeedbackPage } from "@/lib/api";
import { dayTime, n } from "../format";

export const metadata: Metadata = { title: "Feedback" };

const PATH = "/dashboard/admin/feedback";
const one = (v: string | string[] | undefined) => (typeof v === "string" ? v : "");

export default async function FeedbackList({ searchParams }: PageProps<"/dashboard/admin/feedback">) {
  const sp = await searchParams;
  const asked = /^\d{1,15}$/.test(one(sp.offset)) ? Number(one(sp.offset)) : 0;
  let page: AdminFeedbackPage;
  try {
    page = await api.adminFeedback(asked);
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) redirect("/auth/expired");
    if (e instanceof ApiError && e.status === 404) notFound();
    throw e;
  }
  const { feedback, total, offset, page_size: size } = page;
  const href = (to: number) => (to > 0 ? `${PATH}?offset=${to}` : PATH);

  return (
    <>
      <div className="app-head">
        <h1>Feedback</h1>
        <p className="muted small">{n(total)} {total === 1 ? "message" : "messages"}, newest first</p>
      </div>
      <Card aria-label="Messages">
        {feedback.length > 0 ? (
          <Table>
            <thead>
              <tr><th>Date</th><th>From</th><th>Page</th><th>Message</th></tr>
            </thead>
            <tbody>
              {feedback.map((f) => (
                <tr key={f.id}>
                  <td className="small">{dayTime(f.created_at)}</td>
                  <td>
                    {f.user_id ? <Link href={`/dashboard/admin/users/${f.user_id}`}>{f.email || "Account"}</Link> : f.email || <span className="muted">anonymous</span>}
                  </td>
                  <td>{f.page ? <code>{f.page}</code> : <span className="muted">None</span>}</td>
                  <td className="wrap">{f.message}</td>
                </tr>
              ))}
            </tbody>
          </Table>
        ) : total > 0 ? (
          <p className="muted">Nothing on this page. <Link href={href(0)}>Go to the first page</Link></p>
        ) : (
          <p className="muted">No feedback yet.</p>
        )}
        {feedback.length > 0 && total > size && (
          <div className="row">
            <span className="muted small">{n(offset + 1)} to {n(offset + feedback.length)} of {n(total)}</span>
            <span className="inline">
              {offset > 0 && <Link href={href(Math.max(0, offset - size))} rel="prev">Newer</Link>}
              {offset + feedback.length < total && <Link href={href(offset + size)} rel="next">Older</Link>}
            </span>
          </div>
        )}
      </Card>
    </>
  );
}

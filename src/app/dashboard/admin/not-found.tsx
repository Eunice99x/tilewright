import Link from "next/link";

export default function AdminNotFound() {
  return (
    <>
      <h1>Nothing here</h1>
      <p className="muted">No such account or page. <Link href="/dashboard/admin/users">All users</Link></p>
    </>
  );
}

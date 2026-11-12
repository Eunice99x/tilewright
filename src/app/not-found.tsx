import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = { title: "Page not found", robots: { index: false } };

export default function NotFound() {
  return (
    <div className="frame app">
      <h1>Page not found</h1>
      <p className="muted">That page doesn&rsquo;t exist. <Link href="/">Go home</Link></p>
    </div>
  );
}

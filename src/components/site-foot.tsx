"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { APP } from "@/lib/plans";

export function SiteFoot() {
  if (usePathname() === "/") return null;
  return (
    <footer className="wrap foot">
      <span>{APP.name}</span>
      <nav aria-label="Footer" className="inline">
        <Link href="/terms">Terms</Link>
        <Link href="/privacy">Privacy</Link>
      </nav>
    </footer>
  );
}

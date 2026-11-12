import type { Metadata } from "next";
import { LegalPage } from "@/components/legal-page";
import { refunds } from "@/content/legal";

export const metadata: Metadata = { title: refunds.title, description: refunds.description, alternates: { canonical: "/refunds" } };

export default function Page() {
  return <LegalPage doc={refunds} />;
}

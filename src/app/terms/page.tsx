import type { Metadata } from "next";
import { LegalPage } from "@/components/legal-page";
import { terms } from "@/content/legal";

export const metadata: Metadata = { title: terms.title, description: terms.description, alternates: { canonical: "/terms" } };

export default function Page() {
  return <LegalPage doc={terms} />;
}

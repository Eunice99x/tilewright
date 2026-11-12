import type { Metadata } from "next";
import { LegalPage } from "@/components/legal-page";
import { privacy } from "@/content/legal";

export const metadata: Metadata = { title: privacy.title, description: privacy.description, alternates: { canonical: "/privacy" } };

export default function Page() {
  return <LegalPage doc={privacy} />;
}

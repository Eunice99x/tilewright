import { APP } from "@/lib/plans";

export type LegalBlock = string | string[];

export interface LegalDoc {
  slug: "terms" | "privacy" | "refunds";
  title: string;
  description: string;
  intro: string;
  sections: { heading: string; blocks: LegalBlock[] }[];
}

export const terms: LegalDoc = {
  slug: "terms",
  title: "Terms of service",
  description: `TODO: the terms for using ${APP.name}.`,
  intro: "TODO: who you are, what the service does, and that using it means accepting these terms.",
  sections: [
    { heading: "The service", blocks: ["TODO: what you provide and what you don't promise."] },
    { heading: "Accounts", blocks: ["TODO: who may sign up, and what they're responsible for."] },
    { heading: "Billing", blocks: ["TODO: Paddle is the merchant of record. Plans, renewal and cancelling."] },
    { heading: "Contact", blocks: [`TODO: legal entity and address. Email ${APP.support_email}.`] },
  ],
};

export const privacy: LegalDoc = {
  slug: "privacy",
  title: "Privacy policy",
  description: `TODO: what ${APP.name} collects and why.`,
  intro: "TODO: who is responsible for the data.",
  sections: [
    { heading: "What we collect", blocks: [["TODO: account email", "TODO: billing data held by Paddle", "TODO: anything the product stores"]] },
    { heading: "Who we share it with", blocks: [["TODO: hosting (Vercel, Neon)", "TODO: payments (Paddle)", "TODO: email (Resend)"]] },
    { heading: "Your rights", blocks: [`TODO: access and deletion. Email ${APP.support_email}.`] },
  ],
};

export const refunds: LegalDoc = {
  slug: "refunds",
  title: "Refund policy",
  description: `TODO: how refunds work for ${APP.name}.`,
  intro: "TODO: the refund window, and that Paddle processes refunds as merchant of record.",
  sections: [{ heading: "How to ask", blocks: [`TODO: steps. Email ${APP.support_email}.`] }],
};

export const legalDocs = [terms, privacy, refunds];

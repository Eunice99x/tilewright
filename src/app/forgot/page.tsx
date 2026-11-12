import type { Metadata } from "next";
import Link from "next/link";
import { ForgotForm } from "@/components/email-forms";

export const metadata: Metadata = { title: "Reset your password", robots: { index: false } };

export default function Forgot() {
  return (
    <div className="frame auth-wrap">
      <div className="auth card">
        <h1>Reset your password</h1>
        <p className="muted">We&rsquo;ll email you a link.</p>
        <ForgotForm />
        <p className="alt"><Link href="/login">Back to log in</Link></p>
      </div>
    </div>
  );
}

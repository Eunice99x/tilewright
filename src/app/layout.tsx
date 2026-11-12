import type { Metadata, Viewport } from "next";
import { Epilogue } from "next/font/google";
import Link from "next/link";
import { HeaderNav } from "@/components/header-nav";
import { Logo } from "@/components/logo";
import { SiteFoot } from "@/components/site-foot";
import { siteUrl } from "@/lib/oauth";
import { APP } from "@/lib/plans";
import "./globals.css";

const epilogue = Epilogue({ subsets: ["latin"], variable: "--font-epilogue", display: "swap" });

export const viewport: Viewport = { themeColor: "#0d060b" };

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl()),
  title: { default: APP.name, template: `%s | ${APP.name}` },
  description: `${APP.name} is a word-tile game you play with friends: take turns building words on an 11 by 11 board.`,
  openGraph: { type: "website", siteName: APP.name },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={epilogue.variable} data-radius="soft" data-buttons="solid">
      <body suppressHydrationWarning>
        <header className="nav">
          <div className="wrap">
            <a className="skip-link" href="#top">Skip to content</a>
            <Link href="/" className="logo" aria-label={`${APP.name} home`}>
              <Logo />
            </Link>
            <HeaderNav />
          </div>
        </header>
        <main id="top">{children}</main>
        <SiteFoot />
      </body>
    </html>
  );
}

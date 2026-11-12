"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, useSyncExternalStore } from "react";
import { buttonClass } from "@/components/ui/button";

const subscribe = () => () => {};
const loggedIn = () => /(?:^|;\s*)signed_in=1/.test(document.cookie);

const OWN_BUTTON = /^\/(play|join|dashboard)(\/|$)/;

export function HeaderNav() {
  const path = usePathname();
  const inside = useSyncExternalStore(subscribe, loggedIn, () => false);
  const [open, setOpen] = useState(false);
  const home = path === "/";
  const showStart = !OWN_BUTTON.test(path);
  const account = inside
    ? { href: "/dashboard", label: "Your games" }
    : { href: "/login", label: "Log in" };

  return (
    <>
      <nav className="nav-words" aria-label="Main">
        {home && (
          <>
            <div className="mega">
              <a className="nav-word" href="#rules">Rules</a>
              <div className="mega-panel">
                <div className="wrap mega-grid">
                  <div className="mega-col">
                    <span className="label">How to play</span>
                    <a href="#rules">How it works</a>
                    <a href="#bonus">Bonus squares</a>
                    <a href="#numbers">The numbers</a>
                  </div>
                  <div className="mega-col">
                    <span className="label">Ways to play</span>
                    <Link href="/play">With a friend, by link</Link>
                    <Link href="/play">Solo, in 15 turns</Link>
                  </div>
                  <a className="mega-card" href="#rules">
                    <img src="/art/turn-score.webp" width={820} height={614} alt="" />
                    <span><strong>See a turn played</strong><span className="muted">Place, score, pass it back.</span></span>
                  </a>
                </div>
              </div>
            </div>
            <a className="nav-word" href="#bonus">Bonuses</a>
            <a className="nav-word" href="#numbers">Numbers</a>
          </>
        )}
        <Link className="nav-word" href={account.href} aria-current={path === account.href ? "page" : undefined}>{account.label}</Link>
      </nav>
      <div className="nav-actions">
        {showStart && <Link href="/play" className={`${buttonClass()} btn--nav`}>Start a game</Link>}
        <button type="button" className="btn ghost nav-menu-btn" aria-expanded={open} aria-controls="nav-sheet" onClick={() => setOpen((o) => !o)}>
          {open ? "Close" : "Menu"}
        </button>
      </div>
      {open && (
        <div className="sheet" id="nav-sheet" onClick={() => setOpen(false)}>
          {home && (
            <>
              <a href="#rules">How it works</a>
              <a href="#bonus">Bonus squares</a>
              <a href="#numbers">The numbers</a>
            </>
          )}
          <Link href={account.href}>{account.label}</Link>
          {showStart && <Link href="/play" className={buttonClass()}>Start a game</Link>}
        </div>
      )}
    </>
  );
}

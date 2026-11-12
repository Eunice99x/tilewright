"use client";

import Script from "next/script";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { APP } from "@/lib/plans";

interface PaddleGlobal {
  Initialize(o: { token: string }): void;
  Checkout: { open(o: unknown): void };
}
declare global {
  interface Window { Paddle?: PaddleGlobal; __paddleReady?: boolean }
}

export function PaddleButton({ token, priceId, userId, email, label }: { token: string; priceId: string; userId: string; email: string; label: string }) {
  const [ready, setReady] = useState(false);
  function init() {
    if (window.Paddle && !window.__paddleReady) {
      window.Paddle.Initialize({ token });
      window.__paddleReady = true;
    }
    setReady(!!window.Paddle);
  }
  function open() {
    window.Paddle?.Checkout.open({
      items: [{ priceId, quantity: 1 }],
      customer: { email },
      customData: { user_id: userId, product: APP.product },
      settings: { successUrl: `${location.origin}/dashboard/billing?done=1` },
    });
  }
  return (
    <>
      <Script src="https://cdn.paddle.com/paddle/v2/paddle.js" onReady={init} />
      <Button type="button" onClick={open} disabled={!ready}>{label}</Button>
    </>
  );
}

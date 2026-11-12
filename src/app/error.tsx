"use client";

import { Button } from "@/components/ui/button";

export default function RootError({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="frame app" role="alert">
      <h1>Something went wrong</h1>
      <p className="muted">Try again in a moment.</p>
      <div><Button onClick={reset}>Try again</Button></div>
    </div>
  );
}

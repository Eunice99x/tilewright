"use client";

import { Button } from "@/components/ui/button";

export default function AdminError({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="card" role="alert">
      <h1 className="h3">Couldn&rsquo;t load this page</h1>
      <p className="muted">The API may be briefly unavailable.</p>
      <div><Button variant="ghost" onClick={reset}>Try again</Button></div>
    </div>
  );
}

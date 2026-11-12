"use client";

import { Button } from "@/components/ui/button";

export default function DashboardError({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="card" role="alert">
      <h1 className="h3">Couldn&rsquo;t load this page</h1>
      <p className="muted">The service may be briefly unavailable.</p>
      <div><Button onClick={reset}>Try again</Button></div>
    </div>
  );
}

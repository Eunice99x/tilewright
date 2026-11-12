import type { ComponentProps } from "react";

export function Badge({ tone = "neutral", className, ...props }: ComponentProps<"span"> & { tone?: "neutral" | "good" | "bad" }) {
  return <span {...props} className={`badge ${tone}${className ? ` ${className}` : ""}`} />;
}

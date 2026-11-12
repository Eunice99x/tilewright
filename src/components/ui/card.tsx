import type { ComponentProps } from "react";

export function Card({ title, className, children, ...props }: ComponentProps<"section"> & { title?: string }) {
  return (
    <section {...props} className={className ? `card ${className}` : "card"}>
      {title && <h2 className="h3">{title}</h2>}
      {children}
    </section>
  );
}

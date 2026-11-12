import type { ComponentProps } from "react";

export function Table({ className, ...props }: ComponentProps<"table">) {
  return (
    <div className="scroll">
      <table {...props} className={className ? `tbl ${className}` : "tbl"} />
    </div>
  );
}

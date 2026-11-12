import type { ComponentProps } from "react";

export type ButtonVariant = "primary" | "ghost" | "danger";

export function buttonClass(variant: ButtonVariant = "primary"): string {
  return variant === "primary" ? "btn" : `btn ${variant}`;
}

export function Button({ variant, className, ...props }: ComponentProps<"button"> & { variant?: ButtonVariant }) {
  return <button {...props} className={className ? `${buttonClass(variant)} ${className}` : buttonClass(variant)} />;
}

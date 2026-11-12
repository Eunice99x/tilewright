import { APP } from "@/lib/plans";

export function Logo() {
  return (
    <>
      <svg className="mark" viewBox="0 0 32 32" aria-hidden="true">
        <rect x="1" y="1" width="18" height="18" rx="4" fill="#55233f" />
        <rect x="7" y="7" width="18" height="18" rx="4" fill="#85224f" />
        <rect x="13" y="13" width="18" height="18" rx="4" fill="#ff488b" />
      </svg>
      <span>{APP.name}</span>
    </>
  );
}

import Link from "next/link";
import { type LegalDoc, legalDocs } from "@/content/legal";

export function LegalPage({ doc }: { doc: LegalDoc }) {
  const others = legalDocs.filter((d) => d.slug !== doc.slug);
  return (
    <article className="frame legal">
      <p className="notice" role="note">TODO: placeholder text. Have this reviewed and replace it before launch.</p>
      <h1>{doc.title}</h1>
      <p className="muted">{doc.intro}</p>
      {doc.sections.map((s) => (
        <section key={s.heading}>
          <h2>{s.heading}</h2>
          {s.blocks.map((b, i) =>
            Array.isArray(b) ? (
              <ul key={i}>
                {b.map((li) => (
                  <li key={li}>{li}</li>
                ))}
              </ul>
            ) : (
              <p key={i}>{b}</p>
            ),
          )}
        </section>
      ))}
      <nav aria-label="Other policies" className="legal-nav">
        {others.map((o) => (
          <Link key={o.slug} href={`/${o.slug}`}>
            {o.title}
          </Link>
        ))}
      </nav>
    </article>
  );
}

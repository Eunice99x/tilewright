import type { CSSProperties } from "react";
import type { Metadata } from "next";
import Link from "next/link";
import Script from "next/script";
import { buttonClass } from "@/components/ui/button";
import { APP } from "@/lib/plans";
import { ALL_TILES_BONUS, BONUS, RACK_SIZE, SIZE, TILE_COUNT } from "@/lib/rules";

export const metadata: Metadata = { alternates: { canonical: "/" } };

const H1 = `${APP.name}, a word game for two`;

function Rise({ text }: { text: string }) {
  const words = text.split(" ");
  return (
    <>
      {words.map((w, i) => (
        <span key={i}>
          <span className="rise"><span style={{ "--d": `${Math.min(i * 40, 240)}ms` } as CSSProperties}>{w}</span></span>
          {i < words.length - 1 ? " " : null}
        </span>
      ))}
    </>
  );
}

const after = (ms: number) => ({ "--d": `${ms}ms` }) as CSSProperties;

const LEGEND = (["d", "t", "D", "T"] as const).map((b) => ({ b, ...BONUS[b] }));

export default function Home() {
  return (
    <>
      <section className="sec hero-photo" data-enter aria-labelledby="hero-h">
        <picture className="layer hero__bg">
          <source media="(max-width: 1023px)" srcSet="/art/key-art-phone.webp" />
          <img className="layer" src="/art/key-art.webp" width={2880} height={1800} alt="" aria-hidden="true" fetchPriority="high" data-source="client" data-subject="the game's board" />
        </picture>
        <div className="wrap">
          <div className="hero__words hero-text">
            <h1 id="hero-h" aria-label={H1}><Rise text={H1} /></h1>
            <p className="lead after" style={after(420)}>
              Take turns with a friend on one board. Send them a link, play your word whenever you have a minute, and the higher score wins.
            </p>
            <p className="fact after" style={after(480)}>{TILE_COUNT} tiles, two players. Free to play, no account needed.</p>
            <div className="btn-row after" style={after(540)}>
              <Link href="/play" className={buttonClass()}>Start a game</Link>
              <a href="#rules" className={buttonClass("ghost")}>See the rules</a>
            </div>
          </div>
        </div>
      </section>

      <section className="sec" id="rules" aria-labelledby="rules-h">
        <div className="wrap">
          <h2 id="rules-h" className="h2">How it works</h2>
          <p className="sec-line">You and your friend share one board and take turns whenever you both have a minute.</p>
          <div className="cards3">
            <article className="card-open">
              <img className="shot" src="/art/turn-place.webp" width={820} height={614} alt="Three tiles placed under the H of WRIGHT, spelling HYMN down, not played yet" loading="lazy" />
              <h3>Place two or more tiles</h3>
              <p>Lay them in one row or column, joined to a word already on the board.</p>
            </article>
            <article className="card-open">
              <img className="shot" src="/art/turn-score.webp" width={820} height={614} alt="HYMN played down from WRIGHT, worth 22 points" loading="lazy" />
              <h3>Score the words on the board</h3>
              <p>Letters count their number, and bonus squares multiply the tiles you just placed.</p>
            </article>
            <article className="card-open">
              <img className="shot" src="/art/turn-pass.webp" width={820} height={614} alt="A rack of seven tiles, and the line Sam's turn" loading="lazy" />
              <h3>Pass the board back</h3>
              <p>Your rack refills from the bag, and your friend gets the next move.</p>
            </article>
          </div>
        </div>
      </section>

      <section className="sec" id="bonus" aria-labelledby="bonus-h">
        <div className="wrap split">
          <div className="text">
            <h2 id="bonus-h" className="h2">Bonus squares on the board</h2>
            <p className="sec-line">
              A J on a double letter is worth 18. Put the X on a double word and JINX scores 58. Bonuses count once, on the turn you cover them.
            </p>
            <p className="sum">(9×2 + 1 + 1 + 9) × 2 = <strong>58</strong></p>
            <ul className="legend">
              {LEGEND.map(({ b, short, name }) => (
                <li key={b}><span className="swatch" data-bonus={b}>{short}</span>{name}</li>
              ))}
            </ul>
          </div>
          <div className="visual">
            <img className="board-shot" src="/art/bonus-board.webp" width={1256} height={1256} alt={`The ${SIZE} by ${SIZE} board's bonus squares, with JINX across a double letter and a double word`} loading="lazy" />
          </div>
        </div>
      </section>

      <section className="sec" id="numbers" aria-labelledby="numbers-h">
        <div className="wrap split">
          <div className="text">
            <h2 id="numbers-h" className="h2">Tiles and words in numbers</h2>
            <p className="sec-line">Every word is checked against the list before it scores.</p>
            <div className="figs" data-moment="counter-rolls">
              <p className="fig"><span className="t-stat">141,047</span> <span className="cap">words in the list</span></p>
              <p className="fig"><span className="t-stat">{SIZE * SIZE}</span> <span className="cap">squares on the board</span></p>
              <p className="fig"><span className="t-stat">{RACK_SIZE}</span> <span className="cap">tiles on your rack</span></p>
              <p className="fig"><span className="t-stat">{ALL_TILES_BONUS}</span> <span className="cap">extra points for using all {RACK_SIZE}</span></p>
            </div>
          </div>
          <div className="visual">
            <img className="rack-shot" src="/art/rack-friends.webp" width={1256} height={206} alt="A rack of seven tiles spelling FRIENDS" />
          </div>
        </div>
      </section>

      <section className="sec closing" id="start" aria-labelledby="close-h">
        <div className="wrap">
          <div className="close-row">
            <div className="close-text">
              <h2 id="close-h" className="h2">Send a friend a link to the board</h2>
              <p className="sec-line">They join with just a name, and the game keeps both your racks until you finish.</p>
              <Link href="/play" className={buttonClass()}>Start a game</Link>
            </div>
            <img className="invite-shot" src="/art/invite.webp" width={1200} height={278} alt="The invite box: send this link to a friend, with a Copy link button" loading="lazy" />
          </div>
          <footer className="statement">
            <p className="name" aria-hidden="true">{APP.name}</p>
            <div className="cols">
              <nav aria-label="Rules">
                <span className="label">Rules</span>
                <a href="#rules">How it works</a>
                <a href="#bonus">Bonus squares</a>
                <a href="#numbers">The numbers</a>
              </nav>
              <nav aria-label="Play">
                <span className="label">Play</span>
                <Link href="/play">With a friend</Link>
                <Link href="/play">Solo practice</Link>
                <Link href="/login">Log in</Link>
              </nav>
              <nav aria-label="Legal">
                <span className="label">Legal</span>
                <Link href="/privacy">Privacy</Link>
                <Link href="/terms">Terms</Link>
              </nav>
            </div>
            <div className="legal-row">
              <span>© {new Date().getFullYear()} {APP.name}</span>
              <span>Words from the ENABLE list, public domain.</span>
            </div>
          </footer>
        </div>
      </section>

      <Script id="moments" type="module" src="/assets/moments/moments.js" strategy="afterInteractive" />
    </>
  );
}

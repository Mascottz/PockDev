import Link from "next/link";
import { BrandMark } from "./BrandMark";

const LINKS: { href: string; label: string; external?: boolean }[] = [
  { href: "/", label: "Overview" },
  { href: "/deck", label: "Control Deck" },
  { href: "/brand", label: "Brand Kit" },
  { href: "https://github.com/Mascottz/PockDev", label: "GitHub", external: true },
];

export function SiteFooter() {
  return (
    <footer className="relative z-10 border-t border-moss/70 bg-silt">
      <div className="mx-auto max-w-7xl px-5 py-10 sm:px-8">
        <div className="clip-diamond-cut bg-moss p-[1px]">
          <div className="clip-diamond-cut bg-olive">
            <div className="flex flex-col gap-8 px-7 py-8 sm:flex-row sm:items-center sm:justify-between sm:px-10">
              <div className="flex items-center gap-4">
                <BrandMark size={52} animated />
                <div>
                  <p className="font-display text-xl font-extrabold tracking-tight text-chalk">
                    Pock<span className="text-citron">Dev</span>
                  </p>
                  <p className="mt-1 font-mono text-[10px] uppercase tracking-[0.24em] text-ash">
                    Webhook Shield &amp; Pocket SRE
                  </p>
                </div>
              </div>

              <nav aria-label="Footer">
                <ul className="flex flex-wrap items-center gap-x-2 gap-y-2">
                  {LINKS.map((link) => (
                    <li key={link.label}>
                      {link.external ? (
                        <a
                          href={link.href}
                          target="_blank"
                          rel="noreferrer"
                          className="clip-prism-sm inline-block bg-basalt px-4 py-1.5 font-mono text-[11px] uppercase tracking-[0.18em] text-ash transition-colors hover:bg-citron hover:text-silt"
                        >
                          {link.label}
                        </a>
                      ) : (
                        <Link
                          href={link.href}
                          className="clip-prism-sm inline-block bg-basalt px-4 py-1.5 font-mono text-[11px] uppercase tracking-[0.18em] text-ash transition-colors hover:bg-citron hover:text-silt"
                        >
                          {link.label}
                        </Link>
                      )}
                    </li>
                  ))}
                </ul>
              </nav>
            </div>

            <div className="flex items-center gap-3 border-t border-moss/60 px-7 py-4 sm:px-10">
              <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
                <polygon points="6,1 11,10 1,10" fill="#E05A33" />
              </svg>
              <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-ash">
                Catch every webhook. Fix crashes on WhatsApp.
              </p>
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
}

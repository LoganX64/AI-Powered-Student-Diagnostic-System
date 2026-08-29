import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import {
  BarChart3Icon,
  BrainCircuitIcon,
  TargetIcon,
  UsersIcon,
  LineChartIcon,
  SparklesIcon,
} from "lucide-react";
import { landingPageText } from "@/types/static/landing";
import { HoverEffect } from "@/components/ui/card-hover-effect";

const t = landingPageText;

const featureIcons = [
  <BarChart3Icon className="size-6" />,
  <BrainCircuitIcon className="size-6" />,
  <TargetIcon className="size-6" />,
  <UsersIcon className="size-6" />,
  <LineChartIcon className="size-6" />,
  <SparklesIcon className="size-6" />,
];

function HeroIllustration() {
  return (
    <svg viewBox="0 0 720 480" fill="none" className="w-full max-w-lg">
      <defs>
        <marker id="arrow" markerWidth="8" markerHeight="6" refX="7" refY="3" orient="auto">
          <path d="M0,1 L7,3 L0,5 Z" fill="var(--muted-foreground)" />
        </marker>
      </defs>

      {/* Connecting arrows */}
      <path d="M265,155 Q295,150 325,155" stroke="var(--muted-foreground)" strokeWidth="1.5" strokeDasharray="4,3" markerEnd="url(#arrow)" fill="none" opacity="0.4" />
      <path d="M490,195 Q495,235 490,265" stroke="var(--muted-foreground)" strokeWidth="1.5" strokeDasharray="4,3" markerEnd="url(#arrow)" fill="none" opacity="0.4" />
      <path d="M318,395 Q290,390 268,395" stroke="var(--muted-foreground)" strokeWidth="1.5" strokeDasharray="4,3" markerEnd="url(#arrow)" fill="none" opacity="0.4" />

      {/* Sticky 1 — Create Test (top left) */}
      <path d="M30,60 L260,54 Q268,56 266,210 L32,216 Q24,214 30,60Z" fill="var(--primary)" opacity="0.08" stroke="var(--primary)" strokeWidth="1" />
      <path d="M238,54 L266,54 L266,88 Z" fill="var(--primary)" opacity="0.15" stroke="var(--primary)" strokeWidth="0.8" />
      <g transform="translate(78,78) scale(1.3)">
        <rect x="1" y="3" width="24" height="30" rx="2" fill="none" stroke="var(--primary)" strokeWidth="1.8" />
        <line x1="6" y1="10" x2="20" y2="10" stroke="var(--primary)" strokeWidth="1.2" />
        <line x1="6" y1="15" x2="17" y2="15" stroke="var(--primary)" strokeWidth="1.2" />
        <line x1="6" y1="20" x2="19" y2="20" stroke="var(--primary)" strokeWidth="1.2" />
      </g>
      <text x="160" y="165" textAnchor="middle" fontSize="14" fontWeight="700" fill="var(--primary)" fontFamily="var(--font-heading, monospace)">Create Test</text>
      <text x="160" y="185" textAnchor="middle" fontSize="11" fill="var(--muted-foreground)" fontFamily="var(--font-heading, monospace)">Design your exam</text>

      {/* Sticky 2 — Assign Students (top right) */}
      <path d="M320,60 L550,54 Q558,56 556,210 L322,216 Q314,214 320,60Z" fill="var(--primary)" opacity="0.12" stroke="var(--primary)" strokeWidth="1" />
      <path d="M528,54 L556,54 L556,88 Z" fill="var(--primary)" opacity="0.2" stroke="var(--primary)" strokeWidth="0.8" />
      <g transform="translate(368,78) scale(1.3)">
        <circle cx="11" cy="8" r="7" fill="none" stroke="var(--primary)" strokeWidth="1.8" />
        <path d="M1,30 C1,22 21,22 21,30" fill="none" stroke="var(--primary)" strokeWidth="1.8" />
        <circle cx="25" cy="11" r="5.5" fill="none" stroke="var(--primary)" strokeWidth="1.5" />
      </g>
      <text x="450" y="165" textAnchor="middle" fontSize="14" fontWeight="700" fill="var(--primary)" fontFamily="var(--font-heading, monospace)">Assign Students</text>
      <text x="450" y="185" textAnchor="middle" fontSize="11" fill="var(--muted-foreground)" fontFamily="var(--font-heading, monospace)">Link students to test</text>

      {/* Sticky 3 — Student Submits (bottom right) */}
      <path d="M320,270 L550,264 Q558,266 556,420 L322,426 Q314,424 320,270Z" fill="var(--primary)" opacity="0.16" stroke="var(--primary)" strokeWidth="1" />
      <path d="M528,264 L556,264 L556,298 Z" fill="var(--primary)" opacity="0.25" stroke="var(--primary)" strokeWidth="0.8" />
      <g transform="translate(383,288) scale(1.3)">
        <circle cx="13" cy="13" r="11" fill="none" stroke="var(--primary)" strokeWidth="1.8" />
        <path d="M8,13 L12,17 L19,8" stroke="var(--primary)" strokeWidth="2" fill="none" strokeLinecap="round" />
      </g>
      <text x="450" y="375" textAnchor="middle" fontSize="14" fontWeight="700" fill="var(--primary)" fontFamily="var(--font-heading, monospace)">Student Submits</text>
      <text x="450" y="395" textAnchor="middle" fontSize="11" fill="var(--muted-foreground)" fontFamily="var(--font-heading, monospace)">Answers recorded</text>

      {/* Sticky 4 — SQI Analysis (bottom left) */}
      <path d="M30,270 L260,264 Q268,266 266,420 L32,426 Q24,424 30,270Z" fill="#22c55e" opacity="0.1" stroke="#22c55e" strokeWidth="1.2" />
      <path d="M238,264 L266,264 L266,298 Z" fill="#22c55e" opacity="0.18" stroke="#22c55e" strokeWidth="0.8" />
      <g transform="translate(78,288) scale(1.3)">
        <rect x="0" y="14" width="7" height="14" rx="2" fill="#22c55e" opacity="0.5" />
        <rect x="10" y="7" width="7" height="21" rx="2" fill="#22c55e" opacity="0.7" />
        <rect x="20" y="0" width="7" height="28" rx="2" fill="#22c55e" />
        <path d="M3,12 L13,5 L23,-2" stroke="#22c55e" strokeWidth="1.8" fill="none" strokeLinecap="round" />
      </g>
      <text x="160" y="375" textAnchor="middle" fontSize="14" fontWeight="700" fill="#22c55e" fontFamily="var(--font-heading, monospace)">SQI Analysis</text>
      <text x="160" y="395" textAnchor="middle" fontSize="11" fill="var(--muted-foreground)" fontFamily="var(--font-heading, monospace)">Quality insights</text>
    </svg>
  );
}

export function LandingPage() {
  return (
    <div className="flex flex-col min-h-screen">
      {/* Navbar */}
      <header className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <Link to="/" className="flex items-center gap-2">
            <BarChart3Icon className="size-6 text-primary" />
            <span className="text-lg font-bold">{t.brand}</span>
          </Link>
          <nav className="hidden md:flex items-center gap-6 text-sm text-muted-foreground">
            <a href="#features" className="hover:text-foreground transition-colors">{t.nav.features}</a>
            <Link to="/about" className="hover:text-foreground transition-colors">{t.nav.about}</Link>
          </nav>
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="sm" asChild>
              <Link to="/student-login">{t.nav.studentLogin}</Link>
            </Button>
            <Button size="sm" asChild>
              <Link to="/admin-signup">{t.nav.registerNow}</Link>
            </Button>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="mx-auto flex w-full max-w-6xl flex-col-reverse items-center gap-12 px-4 py-20 md:flex-row md:py-28">
        <div className="flex-1 space-y-6">
          <div className="inline-block rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
            {t.hero.badge}
          </div>
          <h1 className="text-4xl font-bold leading-tight tracking-tight md:text-5xl">
            {t.hero.headingLine1}
            <br />
            <span className="text-primary">{t.hero.headingLine2}</span>
          </h1>
          <p className="max-w-lg text-muted-foreground text-base leading-relaxed">
            {t.hero.description}
          </p>
          <div className="flex items-center gap-4">
            <Button size="lg" asChild>
              <Link to="/admin-signup">{t.hero.ctaPrimary}</Link>
            </Button>
            <Button variant="outline" size="lg" asChild>
              <Link to="/student-login">{t.hero.ctaSecondary}</Link>
            </Button>
          </div>
        </div>
        <div className="flex-1 flex justify-center">
          <HeroIllustration />
        </div>
      </section>

      {/* Features */}
      <section id="features" className="border-t bg-muted/40">
        <div className="mx-auto max-w-6xl px-4 py-20">
          <div className="mb-12 text-center">
            <h2 className="text-3xl font-bold tracking-tight">{t.features.sectionTitle}</h2>
            <p className="mt-2 text-muted-foreground">
              {t.features.sectionDescription}
            </p>
          </div>
          <HoverEffect
            className="mt-12"
            items={t.features.items.map((feature, index) => ({
              title: feature.title,
              description: feature.description,
              icon: featureIcons[index],
            }))}
          />
        </div>
      </section>

      {/* CTA */}
      <section className="mx-auto w-full max-w-6xl px-4 py-20">
        <div className="rounded-2xl border bg-card p-8 text-center shadow-sm md:p-12">
          <h2 className="text-2xl font-bold tracking-tight md:text-3xl">
            {t.cta.heading}
          </h2>
          <p className="mx-auto mt-3 max-w-xl text-muted-foreground">
            {t.cta.description}
          </p>
          <div className="mt-6 flex items-center justify-center gap-4">
            <Button size="lg" asChild>
              <Link to="/admin-signup">{t.cta.buttonText}</Link>
            </Button>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t mt-auto">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 text-xs text-muted-foreground">
          <span>{t.footer.copyright}</span>
          <div className="flex gap-4">
            <Link to="/about" className="hover:text-foreground transition-colors">{t.footer.about}</Link>
            <a href="#" className="hover:text-foreground transition-colors">{t.footer.privacy}</a>
            <a href="#" className="hover:text-foreground transition-colors">{t.footer.terms}</a>
          </div>
        </div>
      </footer>
    </div>
  );
}

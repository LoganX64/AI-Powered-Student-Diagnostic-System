import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import {
  BarChart3Icon,
} from "lucide-react";
import { landingPageText } from "@/types/static/landing";
import { HoverEffect } from "@/components/ui/card-hover-effect";

const t = landingPageText;

const featureIcons = [
  <img src="/images/cartoon-person-appraising-performance.svg" alt="" className="h-16 w-auto" />,
  <img src="/images/person-presenting-concept.svg" alt="" className="h-16 w-auto" />,
  <img src="/images/facilitation-prioritisation.svg" alt="" className="h-16 w-auto" />,
  <img src="/images/role-permissions.svg" alt="" className="h-16 w-auto" />,
  <img src="/images/chart.svg" alt="" className="h-16 w-auto" />,
  <img src="/images/cartoon-ai-model-as-single.svg" alt="" className="h-16 w-auto" />,
];

function HeroIllustration() {
  return (
    <svg viewBox="0 0 800 560" fill="none" className="w-full max-w-lg">
      <defs>
        <marker id="arrow" markerWidth="8" markerHeight="6" refX="7" refY="3" orient="auto">
          <path d="M0,1 L7,3 L0,5 Z" fill="var(--muted-foreground)" />
        </marker>
        <filter id="orangeIcon">
          <feColorMatrix
            type="matrix"
            values="0 0 0 0 0.76
                    0 0 0 0 0.33
                    0 0 0 0 0.18
                    0 0 0 1 0"
          />
        </filter>
        <filter id="greenIcon">
          <feColorMatrix
            type="matrix"
            values="0 0 0 0 0.13
                    0 0 0 0 0.77
                    0 0 0 0 0.37
                    0 0 0 1 0"
          />
        </filter>
      </defs>

      {/* Connecting arrows */}
      <path d="M295,180 Q335,172 375,180" stroke="var(--muted-foreground)" strokeWidth="1.5" strokeDasharray="4,3" markerEnd="url(#arrow)" fill="none" opacity="0.4" />
      <path d="M560,230 Q565,280 560,320" stroke="var(--muted-foreground)" strokeWidth="1.5" strokeDasharray="4,3" markerEnd="url(#arrow)" fill="none" opacity="0.4" />
      <path d="M365,460 Q330,452 295,460" stroke="var(--muted-foreground)" strokeWidth="1.5" strokeDasharray="4,3" markerEnd="url(#arrow)" fill="none" opacity="0.4" />

      {/* Sticky 1 — Create Test (top left) */}
      <path d="M20,50 L310,44 Q320,46 318,250 L22,256 Q12,254 20,50Z" fill="white" stroke="var(--primary)" strokeWidth="1" />
      <path d="M288,44 L318,44 L318,84 Z" fill="var(--primary)" stroke="var(--primary)" strokeWidth="0.8" />
      <image href="/images/cartoon-test-suite-list.svg" x="100" y="70" width="100" height="100" filter="url(#orangeIcon)" />
      <text x="180" y="205" textAnchor="middle" fontSize="15" fontWeight="700" fill="var(--primary)" fontFamily="var(--font-heading, monospace)">Create Test</text>
      <text x="180" y="228" textAnchor="middle" fontSize="12" fill="var(--muted-foreground)" fontFamily="var(--font-heading, monospace)">Design your exam</text>

      {/* Sticky 2 — Assign Students (top right) */}
      <path d="M380,50 L670,44 Q680,46 678,250 L382,256 Q372,254 380,50Z" fill="white" stroke="var(--primary)" strokeWidth="1" />
      <path d="M648,44 L678,44 L678,84 Z" fill="var(--primary)" stroke="var(--primary)" strokeWidth="0.8" />
      <image href="/images/cartoon-person-assigning-owner.svg" x="460" y="70" width="100" height="100" filter="url(#orangeIcon)" />
      <text x="540" y="205" textAnchor="middle" fontSize="15" fontWeight="700" fill="var(--primary)" fontFamily="var(--font-heading, monospace)">Assign Students</text>
      <text x="540" y="228" textAnchor="middle" fontSize="12" fill="var(--muted-foreground)" fontFamily="var(--font-heading, monospace)">Link students to test</text>

      {/* Sticky 3 — Student Submits (bottom right) */}
      <path d="M380,310 L670,304 Q680,306 678,510 L382,516 Q372,514 380,310Z" fill="white" stroke="var(--primary)" strokeWidth="1" />
      <path d="M648,304 L678,304 L678,344 Z" fill="var(--primary)" stroke="var(--primary)" strokeWidth="0.8" />
      <image href="/images/cartoon-person-submitting-essay.svg" x="460" y="330" width="100" height="100" filter="url(#orangeIcon)" />
      <text x="540" y="465" textAnchor="middle" fontSize="15" fontWeight="700" fill="var(--primary)" fontFamily="var(--font-heading, monospace)">Student Submits</text>
      <text x="540" y="488" textAnchor="middle" fontSize="12" fill="var(--muted-foreground)" fontFamily="var(--font-heading, monospace)">Answers recorded</text>

      {/* Sticky 4 — SQI Analysis (bottom left) */}
      <path d="M20,310 L310,304 Q320,306 318,510 L22,516 Q12,514 20,310Z" fill="white" stroke="#22c55e" strokeWidth="1.2" />
      <path d="M288,304 L318,304 L318,344 Z" fill="#22c55e" stroke="#22c55e" strokeWidth="0.8" />
      <image href="/images/cartoon-dashboard-for-insights.svg" x="100" y="330" width="100" height="100" filter="url(#greenIcon)" />
      <text x="180" y="465" textAnchor="middle" fontSize="15" fontWeight="700" fill="#22c55e" fontFamily="var(--font-heading, monospace)">SQI Analysis</text>
      <text x="180" y="488" textAnchor="middle" fontSize="12" fill="var(--muted-foreground)" fontFamily="var(--font-heading, monospace)">Quality insights</text>
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

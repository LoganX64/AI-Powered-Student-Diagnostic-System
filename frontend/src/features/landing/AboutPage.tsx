import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { BarChart3Icon, MailIcon, MapPinIcon, BuildingIcon } from "lucide-react";
import { aboutPageText } from "@/types/static/about";
import { HoverEffect } from "@/components/ui/card-hover-effect";
import { SvgIcon } from "@/components/ui/svg-icon";

const t = aboutPageText;

const aboutIcons = [
  <SvgIcon src="/images/isometric-bank.svg" className="h-16 w-auto text-primary" />,
  <SvgIcon src="/images/coach-clipboard.svg" className="h-16 w-auto text-primary" />,
  <SvgIcon src="/images/student-backpack.svg" className="h-16 w-auto text-primary" />,
  <SvgIcon src="/images/parent-child.svg" className="h-16 w-auto text-primary" />,
];

export function AboutPage() {
  return (
    <div className="flex flex-col min-h-screen">
      {/* Navbar */}
      <header className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <Link to="/" className="flex items-center gap-2">
            <BarChart3Icon className="size-6 text-primary" />
            <span className="text-lg font-bold">EduQuant</span>
          </Link>
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="sm" asChild>
              <Link to="/student-login">Student Login</Link>
            </Button>
            <Button size="sm" asChild>
              <Link to="/admin-signup">Register Now</Link>
            </Button>
          </div>
        </div>
      </header>

      {/* Content */}
      <main className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8 py-16 space-y-16">
        {/* Hero */}
        <div className="text-center space-y-4 max-w-3xl mx-auto">
          <h1 className="text-4xl font-bold tracking-tight">{t.hero.title}</h1>
          <p className="text-muted-foreground text-lg leading-relaxed">
            {t.hero.description}
          </p>
        </div>

        {/* Mission */}
        <section className="rounded-xl border bg-card p-8 md:p-10 shadow-sm">
          <h2 className="text-2xl font-semibold mb-4">{t.mission.title}</h2>
          <p className="text-muted-foreground leading-relaxed text-base">
            {t.mission.description}
          </p>
        </section>

        {/* What We Do */}
        <section>
          <h2 className="text-2xl font-semibold mb-6">{t.whatWeDo.title}</h2>
          <HoverEffect
            columns={{ lg: 4 }}
            items={t.whatWeDo.items.map((item, index) => ({
              title: item.title,
              description: item.description,
              icon: aboutIcons[index],
            }))}
          />
        </section>

        {/* Contact */}
        <section className="rounded-xl border bg-card p-8 md:p-10 shadow-sm">
          <h2 className="text-2xl font-semibold mb-6">{t.contact.title}</h2>
          <div className="grid gap-8 sm:grid-cols-3">
            <div className="flex items-start gap-4">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <BuildingIcon className="size-5" />
              </div>
              <div>
                <p className="text-sm font-medium mb-1">{t.contact.company.company}</p>
                <p className="text-sm text-muted-foreground">EduQuant</p>
              </div>
            </div>
            <div className="flex items-start gap-4">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <MailIcon className="size-5" />
              </div>
              <div>
                <p className="text-sm font-medium mb-1">{t.contact.company.helpline}</p>
                <a
                  href={`mailto:${t.contact.company.helplineEmail}`}
                  className="text-sm text-muted-foreground hover:text-foreground transition-colors break-all"
                >
                  {t.contact.company.helplineEmail}
                </a>
              </div>
            </div>
            <div className="flex items-start gap-4">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <MapPinIcon className="size-5" />
              </div>
              <div>
                <p className="text-sm font-medium mb-1">{t.contact.company.address}</p>
                <p className="text-sm text-muted-foreground">
                  {t.contact.company.addressFull}
                </p>
              </div>
            </div>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t mt-auto">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8 text-xs text-muted-foreground">
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

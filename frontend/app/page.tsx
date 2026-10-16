import Link from "next/link";

const NAV_LINKS = [
  { href: "#about", label: "About" },
  { href: "#academics", label: "Academics" },
  { href: "#admissions", label: "Admissions" },
  { href: "#news", label: "News" },
  { href: "#contact", label: "Contact" },
];

const STATS = [
  { value: "1,200+", label: "Students enrolled" },
  { value: "85+", label: "Expert faculty" },
  { value: "98%", label: "Graduation rate" },
  { value: "40+", label: "Clubs & sports" },
];

const PROGRAMMES = [
  {
    title: "Primary School",
    years: "Basic 1 – 6",
    blurb:
      "A joyful, structured foundation where curiosity is trained into confident reading, numeracy and inquiry habits.",
  },
  {
    title: "Junior High School (J.H.S)",
    years: "J.H.S 1 – 3",
    blurb:
      "Subject-deep teaching with mentor-led study skills, letting every learner find their strengths before the senior years.",
  },
  {
    title: "Senior High School (S.H.S)",
    years: "S.H.S 1 – 3",
    blurb:
      "Rigorous university-preparatory tracks in Sciences, Humanities and Business with university counselling from day one.",
  },
];

const WHY_US = [
  {
    title: "Academic excellence",
    blurb:
      "Small class sizes, continuous assessment and a track record of top-tier exam results and university placements.",
  },
  {
    title: "Whole-child development",
    blurb:
      "Sports, music, debate and service learning are timetabled — not add-ons — so character grows alongside grades.",
  },
  {
    title: "Safe, modern campus",
    blurb:
      "Secure gated campus with science labs, a library, ICT suites and dedicated pastoral care staff.",
  },
  {
    title: "Real parent partnership",
    blurb:
      "Live attendance, grades and announcements in the parent portal, plus two formal conferencing terms a year.",
  },
];

const NEWS = [
  {
    date: "Oct 6, 2026",
    tag: "Admissions",
    title: "January intake applications now open",
    blurb:
      "Entrance assessments hold every Saturday in November. Places in Form 1 are limited — early application is advised.",
  },
  {
    date: "Oct 2, 2026",
    tag: "Results",
    title: "Record senior examination results",
    blurb:
      "Our latest graduating class posted the school's strongest mathematics and science results in a decade.",
  },
  {
    date: "Sep 25, 2026",
    tag: "Events",
    title: "Inter-house athletics festival returns",
    blurb:
      "The annual sports festival takes place mid-October — families are warmly welcome on the main field.",
  },
];

export default function HomePage() {
  return (
    <div className="min-h-screen bg-white text-slate-800">
      {/* ===== Announcement bar ===== */}
      <div className="bg-brand-900 px-4 py-2 text-center text-xs font-medium text-brand-50">
        Admissions for the 2027 academic year are open —{" "}
        <a href="#admissions" className="underline underline-offset-2 hover:text-white">
          book an entrance assessment
        </a>
      </div>

      {/* ===== Header ===== */}
      <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/90 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-3">
          <Link href="/" className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-700 text-sm font-extrabold text-white">
              SA
            </span>
            <span className="leading-tight">
              <span className="block text-base font-extrabold tracking-tight text-brand-900">
                SchoolHub Academy
              </span>
              <span className="block text-[10px] font-semibold uppercase tracking-widest text-slate-500">
                Learn · Lead · Serve
              </span>
            </span>
          </Link>

          <nav className="hidden items-center gap-7 text-sm font-medium text-slate-600 lg:flex">
            {NAV_LINKS.map((l) => (
              <a key={l.href} href={l.href} className="hover:text-brand-700 transition-colors">
                {l.label}
              </a>
            ))}
          </nav>

          <div className="flex items-center gap-3">
            <Link
              href="/login"
              className="hidden text-sm font-semibold text-brand-700 hover:text-brand-600 sm:inline-block"
            >
              Sign in
            </Link>
            <Link
              href="#admissions"
              className="rounded-lg bg-brand-700 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-brand-600"
            >
              Apply now
            </Link>
          </div>
        </div>
      </header>

      {/* ===== Hero ===== */}
      <section className="relative bg-brand-900">
        <img
          src="https://images.unsplash.com/photo-1580582932707-520aed937b7b?auto=format&fit=crop&w=1600&q=70"
          alt="Students on a school campus"
          className="absolute inset-0 h-full w-full object-cover opacity-30"
        />
        <div className="relative mx-auto max-w-6xl px-6 py-24 lg:py-32">
          <p className="mb-4 inline-block rounded-full border border-white/20 bg-white/10 px-3 py-1 text-xs font-semibold uppercase tracking-widest text-white/90">
            Independent day school
          </p>
          <h1 className="max-w-2xl text-4xl font-extrabold leading-tight tracking-tight text-white sm:text-5xl">
            Where bright futures begin.
          </h1>
          <p className="mt-4 max-w-xl text-base leading-relaxed text-white/80 sm:text-lg">
            SchoolHub Academy pairs rigorous academics with genuine care for every
            child — from the first day of Basic 1 to the university acceptance letter.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <a
              href="#admissions"
              className="rounded-lg bg-white px-6 py-3 text-sm font-semibold text-brand-900 shadow-sm transition-colors hover:bg-brand-50"
            >
              Start your application
            </a>
            <a
              href="#about"
              className="rounded-lg border border-white/30 px-6 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/10"
            >
              Discover the school
            </a>
          </div>
        </div>
      </section>

      {/* ===== Stats band ===== */}
      <section className="border-b border-slate-200 bg-slate-50">
        <div className="mx-auto grid max-w-6xl grid-cols-2 gap-8 px-6 py-10 sm:grid-cols-4">
          {STATS.map((s) => (
            <div key={s.label} className="text-center">
              <p className="text-3xl font-extrabold text-brand-900">{s.value}</p>
              <p className="mt-1 text-xs font-medium uppercase tracking-wide text-slate-500">
                {s.label}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* ===== About ===== */}
      <section id="about" className="mx-auto max-w-6xl scroll-mt-20 px-6 py-20">
        <div className="grid items-center gap-10 lg:grid-cols-2">
          <img
            src="https://images.unsplash.com/photo-1509062522246-3755977927d7?auto=format&fit=crop&w=900&q=70"
            alt="Teacher with students in a classroom"
            className="h-80 w-full rounded-2xl object-cover shadow-md"
          />
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-brand-600">
              About the school
            </p>
            <h2 className="mt-2 text-2xl font-extrabold tracking-tight text-brand-900 sm:text-3xl">
              Thirty years of teaching children to think, not just to remember.
            </h2>
            <p className="mt-4 text-sm leading-relaxed text-slate-600">
              Founded in 1996, SchoolHub Academy blends a demanding academic core
              with small-class attention and a strong pastoral system. Our teachers
              are subject specialists who know every child by name — and our
              alumni go on to leading universities and professions across the region.
            </p>
            <p className="mt-3 text-sm leading-relaxed text-slate-600">
              Everything we do is built on three commitments: learning that lasts,
              leadership with humility, and service beyond ourselves.
            </p>
            <a
              href="#academics"
              className="mt-6 inline-flex items-center gap-1 text-sm font-semibold text-brand-700 hover:text-brand-600"
            >
              Explore our programmes <span aria-hidden>&rarr;</span>
            </a>
          </div>
        </div>
      </section>

      {/* ===== Academics ===== */}
      <section id="academics" className="scroll-mt-20 bg-slate-50 py-20">
        <div className="mx-auto max-w-6xl px-6">
          <div className="max-w-2xl">
            <p className="text-xs font-semibold uppercase tracking-widest text-brand-600">
              Academics
            </p>
            <h2 className="mt-2 text-2xl font-extrabold tracking-tight text-brand-900 sm:text-3xl">
              One school. Three stages. A continuous journey.
            </h2>
          </div>
          <div className="mt-10 grid gap-6 md:grid-cols-3">
            {PROGRAMMES.map((p) => (
              <div
                key={p.title}
                className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm transition-shadow hover:shadow-md"
              >
                <p className="text-xs font-semibold uppercase tracking-wide text-brand-600">
                  {p.years}
                </p>
                <h3 className="mt-2 text-lg font-bold text-brand-900">{p.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-600">{p.blurb}</p>
              </div>
            ))}
          </div>

          <div className="mt-14 grid gap-6 md:grid-cols-2">
            {WHY_US.map((w) => (
              <div key={w.title} className="flex gap-4">
                <span className="mt-0.5 flex h-8 w-8 flex-none items-center justify-center rounded-full bg-brand-100 text-sm text-brand-700">
                  ✓
                </span>
                <div>
                  <h3 className="text-sm font-bold text-slate-900">{w.title}</h3>
                  <p className="mt-1 text-sm leading-relaxed text-slate-600">{w.blurb}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ===== Admissions ===== */}
      <section id="admissions" className="mx-auto max-w-6xl scroll-mt-20 px-6 py-20">
        <div className="rounded-2xl bg-brand-900 px-8 py-12 text-center shadow-lg sm:px-16">
          <h2 className="text-2xl font-extrabold tracking-tight text-white sm:text-3xl">
            Join the SchoolHub family
          </h2>
          <p className="mx-auto mt-3 max-w-xl text-sm leading-relaxed text-white/75">
            Applications for the next intake are open. Tell us about your child and
            our admissions team will arrange a campus tour and entrance assessment
            within three working days.
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-3">
            <a
              href="mailto:admissions@schoolhub.test"
              className="rounded-lg bg-white px-6 py-3 text-sm font-semibold text-brand-900 shadow-sm transition-colors hover:bg-brand-50"
            >
              Contact admissions
            </a>
            <a
              href="#contact"
              className="rounded-lg border border-white/30 px-6 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/10"
            >
              Book a campus tour
            </a>
          </div>
        </div>
      </section>

      {/* ===== News ===== */}
      <section id="news" className="scroll-mt-20 bg-slate-50 py-20">
        <div className="mx-auto max-w-6xl px-6">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-widest text-brand-600">
                News &amp; announcements
              </p>
              <h2 className="mt-2 text-2xl font-extrabold tracking-tight text-brand-900 sm:text-3xl">
                What&apos;s happening on campus
              </h2>
            </div>
            <Link
              href="/login"
              className="text-sm font-semibold text-brand-700 hover:text-brand-600"
            >
              Parents &amp; staff: read all notices in the portal →
            </Link>
          </div>
          <div className="mt-10 grid gap-6 md:grid-cols-3">
            {NEWS.map((n) => (
              <article
                key={n.title}
                className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
              >
                <div className="flex items-center gap-2 text-xs">
                  <span className="rounded-full bg-brand-100 px-2.5 py-0.5 font-semibold text-brand-700">
                    {n.tag}
                  </span>
                  <time className="text-slate-500">{n.date}</time>
                </div>
                <h3 className="mt-3 text-sm font-bold leading-snug text-slate-900">
                  {n.title}
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-600">{n.blurb}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      {/* ===== Footer ===== */}
      <footer id="contact" className="scroll-mt-20 border-t border-slate-200 bg-brand-900 text-white">
        <div className="mx-auto grid max-w-6xl gap-10 px-6 py-14 md:grid-cols-4">
          <div className="md:col-span-2">
            <div className="flex items-center gap-2.5">
              <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-white/10 text-sm font-extrabold">
                SA
              </span>
              <span className="text-base font-extrabold tracking-tight">SchoolHub Academy</span>
            </div>
            <p className="mt-4 max-w-sm text-sm leading-relaxed text-white/70">
              An independent day school committed to academic excellence and
              whole-child development. Learn · Lead · Serve.
            </p>
          </div>
          <div>
            <h3 className="text-xs font-semibold uppercase tracking-widest text-white/60">
              Contact
            </h3>
            <ul className="mt-4 space-y-2 text-sm text-white/75">
              <li>123 Independence Avenue</li>
              <li>Accra, Ghana</li>
              <li>
                <a href="tel:+233300000000" className="hover:text-white">
                  +233 30 000 0000
                </a>
              </li>
              <li>
                <a href="mailto:hello@schoolhub.test" className="hover:text-white">
                  hello@schoolhub.test
                </a>
              </li>
            </ul>
          </div>
          <div>
            <h3 className="text-xs font-semibold uppercase tracking-widest text-white/60">
              Quick links
            </h3>
            <ul className="mt-4 space-y-2 text-sm text-white/75">
              <li>
                <a href="#admissions" className="hover:text-white">Admissions</a>
              </li>
              <li>
                <a href="#academics" className="hover:text-white">Academics</a>
              </li>
              <li>
                <Link href="/login" className="hover:text-white">Portal sign-in</Link>
              </li>
              <li>
                <a href="#contact" className="hover:text-white">Contact us</a>
              </li>
            </ul>
          </div>
        </div>
        <div className="border-t border-white/10 py-5 text-center text-xs text-white/50">
          © {new Date().getFullYear()} SchoolHub Academy. All rights reserved.
        </div>
      </footer>
    </div>
  );
}

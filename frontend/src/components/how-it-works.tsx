import { SectionHeader } from "./ui/section-header";

const STEPS = [
  {
    title: "Search in parallel",
    body: "Your query goes to DuckDuckGo, Bing and Yahoo at the same time. A slow engine never holds up the rest.",
  },
  {
    title: "Merge duplicates",
    body: "Redirects and tracking links are cleaned up, so the same page from three engines becomes one result.",
  },
  {
    title: "Rank by agreement",
    body: "Pages that several engines rank highly come first. Off-topic and spam results are dropped.",
  },
];

export function HowItWorks() {
  return (
    <section id="how-it-works" className="w-full min-h-[calc(100vh-3.5rem)] min-h-[calc(100dvh-3.5rem)] py-20 flex flex-col justify-center scroll-mt-14 gap-16 sm:gap-20">
      <SectionHeader eyebrow="How it works" title="Three engines in, one clean list out" />

      <ol className="grid grid-cols-1 md:grid-cols-3 gap-12 md:gap-10">
        {STEPS.map((step, i) => (
          <li key={step.title} className="flex flex-col gap-4">
            <div className="flex items-center gap-3">
              <span className="text-base font-medium text-brand dark:text-brand-light tabular-nums">0{i + 1}</span>
              <span aria-hidden="true" className="h-px flex-1 bg-gradient-to-r from-zinc-300 dark:from-zinc-700 to-transparent" />
            </div>
            <h3 className="text-2xl font-semibold tracking-tight text-zinc-950 dark:text-zinc-50">{step.title}</h3>
            <p className="text-[17px] leading-relaxed text-zinc-600 dark:text-zinc-400">{step.body}</p>
          </li>
        ))}
      </ol>
    </section>
  );
}

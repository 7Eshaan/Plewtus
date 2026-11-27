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
    <section
      id="how-it-works"
      className="flex min-h-[calc(100vh-4rem)] min-h-[calc(100dvh-4rem)] w-full scroll-mt-16 flex-col justify-center gap-12 py-20 sm:gap-16"
    >
      <SectionHeader eyebrow="How it works" title="Three engines in, one clean list out" />

      <ol className="grid grid-cols-1 gap-4 md:grid-cols-3">
        {STEPS.map((step, i) => (
          <li key={step.title} className="flex flex-col gap-4 rounded-corner-xl bg-surface-container-low/80 p-6 backdrop-blur-sm">
            <span
              aria-hidden="true"
              className="flex size-12 items-center justify-center rounded-corner-lg bg-secondary-container text-title-lg font-medium text-on-secondary-container tabular-nums"
            >
              {i + 1}
            </span>
            <h3 className="text-headline-sm text-on-surface">{step.title}</h3>
            <p className="text-body-lg text-on-surface-variant">{step.body}</p>
          </li>
        ))}
      </ol>
    </section>
  );
}

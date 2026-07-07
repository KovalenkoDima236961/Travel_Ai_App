import Link from "next/link";
import { PageContainer } from "@/components/layout/PageContainer";
import { buttonStyles } from "@/shared/ui/button";

export function HomePageContent() {
  return (
    <PageContainer className="py-10">
      <section className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_22rem] lg:items-center">
        <div className="max-w-3xl">
          <p className="text-sm font-semibold uppercase text-primary-700">
            Web App v1
          </p>
          <h1 className="mt-3 text-4xl font-semibold text-slate-950 sm:text-5xl">
            Travel AI Planner
          </h1>
          <p className="mt-5 max-w-2xl text-lg leading-8 text-slate-600">
            Create a trip request, generate an itinerary from the Trip Service,
            and review the result in one focused workspace.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <Link className={buttonStyles()} href="/trips/new">
              Create trip
            </Link>
            <Link className={buttonStyles({ variant: "secondary" })} href="/trips">
              View trips
            </Link>
          </div>
        </div>

        <aside className="rounded-lg border border-slate-200 bg-white p-6 shadow-soft">
          <h2 className="text-base font-semibold text-slate-950">Planning flow</h2>
          <div className="mt-5 space-y-4">
            {[
              ["1", "Create a trip request"],
              ["2", "Generate itinerary"],
              ["3", "Review daily plan"]
            ].map(([step, label]) => (
              <div key={step} className="flex items-center gap-3">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-50 text-sm font-semibold text-primary-700">
                  {step}
                </span>
                <span className="text-sm font-medium text-slate-700">{label}</span>
              </div>
            ))}
          </div>
        </aside>
      </section>
    </PageContainer>
  );
}

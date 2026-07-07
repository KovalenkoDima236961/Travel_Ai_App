import Link from "next/link";
import { PageContainer } from "@/components/layout/PageContainer";
import { buttonStyles } from "@/shared/ui/button";

export function OfflinePageContent() {
  return (
    <PageContainer>
      <section className="rounded-lg border border-amber-200 bg-amber-50 p-6 text-amber-950">
        <h1 className="text-2xl font-semibold">You are offline</h1>
        <p className="mt-3 max-w-2xl text-sm leading-6">
          Open a trip you have previously viewed to see saved data.
        </p>
        <Link className={buttonStyles({ className: "mt-5" })} href="/trips">
          Go to trips
        </Link>
      </section>
    </PageContainer>
  );
}

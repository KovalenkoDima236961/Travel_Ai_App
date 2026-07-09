import { cn } from "@/shared/lib/cn";
import { ClosingCta } from "./ClosingCta";
import { instrumentSans, newsreader } from "./fonts";
import { HowItWorks } from "./HowItWorks";
import { LandingFooter } from "./LandingFooter";
import { LandingHeader } from "./LandingHeader";
import { LandingHero } from "./LandingHero";

export function LandingPage() {
  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen overflow-x-hidden bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <LandingHeader />
      <LandingHero />
      <HowItWorks />
      <ClosingCta />
      <LandingFooter />
    </div>
  );
}

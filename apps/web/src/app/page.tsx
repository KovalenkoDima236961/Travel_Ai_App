import { LandingGate } from "@/_pages/landing/ui/LandingGate";
import { LandingPage } from "@/_pages/landing/ui/LandingPage";

export default function HomePage() {
  return (
    <LandingGate>
      <LandingPage />
    </LandingGate>
  );
}

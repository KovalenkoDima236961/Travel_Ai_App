import { Instrument_Sans, Newsreader } from "next/font/google";

// Scoped to the workspace-detail slice: the CSS variables are applied on the
// screen root wrapper only, matching landing/auth/trips/workspaces so the
// redesigned pages share one type system without adding a global font preload.
export const newsreader = Newsreader({
  subsets: ["latin"],
  style: ["normal", "italic"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-newsreader",
  display: "swap"
});

export const instrumentSans = Instrument_Sans({
  subsets: ["latin"],
  variable: "--font-instrument-sans",
  display: "swap"
});

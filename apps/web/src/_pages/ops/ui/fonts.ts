import { Instrument_Sans, JetBrains_Mono, Newsreader } from "next/font/google";

// Scoped to the ops slice: the CSS variables are applied on the Ops screen root
// wrapper only, matching landing/auth/trips/workspaces so the redesigned pages
// share one type system without adding a global font preload. Ops additionally
// loads JetBrains Mono for its many ID cells (job/message/correlation ids).
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

export const jetBrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["400", "500"],
  variable: "--font-jetbrains-mono",
  display: "swap"
});

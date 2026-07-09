import { Instrument_Sans, Newsreader } from "next/font/google";

// Scoped to the auth slice: the CSS variables are applied on the auth screen root
// wrapper only, matching the marketing landing so the two share one type system.
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

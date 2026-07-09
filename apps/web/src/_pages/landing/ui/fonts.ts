import { Instrument_Sans, Newsreader } from "next/font/google";

// Scoped to the landing slice: the CSS variables are applied on the landing
// root wrapper only, so the rest of the app carries no extra font preload.
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

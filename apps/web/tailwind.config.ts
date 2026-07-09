import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        primary: {
          50: "#eff6ff",
          100: "#dbeafe",
          600: "#2563eb",
          700: "#1d4ed8"
        },
        // Warm editorial palette for the marketing landing page.
        clay: {
          DEFAULT: "#C05B3B",
          dark: "#A84A2E",
          deep: "#8F3D24",
          bright: "#D4693F",
          glow: "#E0885E",
          tint: "#F7E4DB"
        },
        sand: {
          50: "#FAF6F1",
          100: "#FBF3EC",
          150: "#F6EDE2",
          200: "#F1E8DC",
          300: "#EDE4D7",
          400: "#DCCFBE",
          600: "#C4B5A3"
        },
        cocoa: {
          900: "#221A14",
          700: "#4A3F35",
          500: "#6B5D50",
          400: "#8A7A6A"
        }
      },
      fontFamily: {
        newsreader: ["var(--font-newsreader)", "Georgia", "serif"],
        instrument: ["var(--font-instrument-sans)", "system-ui", "sans-serif"]
      },
      boxShadow: {
        soft: "0 12px 30px rgba(15, 23, 42, 0.08)"
      }
    }
  },
  plugins: []
};

export default config;

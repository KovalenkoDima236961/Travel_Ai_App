import type { Metadata, Viewport } from "next";
import { ReactNode } from "react";
import "leaflet/dist/leaflet.css";
import "./globals.css";
import { AppHeader } from "@/components/layout/AppHeader";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "Travel AI Planner",
  applicationName: "Travel AI",
  description: "Plan, edit, and use AI-powered travel itineraries with offline access.",
  manifest: "/manifest.json",
  appleWebApp: {
    capable: true,
    title: "Travel AI",
    statusBarStyle: "default"
  },
  icons: {
    icon: [
      { url: "/icons/icon-192x192.png", sizes: "192x192", type: "image/png" },
      { url: "/icons/icon-512x512.png", sizes: "512x512", type: "image/png" }
    ],
    apple: [{ url: "/icons/icon-152x152.png", sizes: "152x152", type: "image/png" }]
  }
};

export const viewport: Viewport = {
  themeColor: "#2563eb"
};

type RootLayoutProps = {
  children: ReactNode;
};

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <html lang="en">
      <body>
        <Providers>
          <AppHeader />
          <main>{children}</main>
        </Providers>
      </body>
    </html>
  );
}

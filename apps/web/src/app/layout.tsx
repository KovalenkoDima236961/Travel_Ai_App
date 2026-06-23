import type { Metadata } from "next";
import { ReactNode } from "react";
import "leaflet/dist/leaflet.css";
import "./globals.css";
import { AppHeader } from "@/components/layout/AppHeader";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "Travel AI Planner",
  description: "Create AI-assisted travel plans and generate itineraries."
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

import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

// Liveness deliberately avoids upstream calls: a running web process is alive
// even when a backend dependency is temporarily unavailable.
export function GET() {
  return NextResponse.json({ status: "ok", service: "web-app" });
}

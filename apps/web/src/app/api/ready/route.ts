import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

// Compose orders the web container after its required APIs. This endpoint is a
// lightweight readiness contract for the web runtime itself; browser API calls
// remain independently resilient to an upstream restart.
export function GET() {
  return NextResponse.json({ status: "ready", service: "web-app", dependencies: {} });
}

import { NextResponse } from "next/server";
import { webVersionInfo } from "@/lib/version";

export function GET() {
  return NextResponse.json(webVersionInfo);
}

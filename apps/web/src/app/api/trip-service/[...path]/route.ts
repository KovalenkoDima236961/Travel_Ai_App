import { NextRequest, NextResponse } from "next/server";
import { getTripServiceUrl } from "@/lib/config";

type RouteContext = {
  params: Promise<{
    path: string[];
  }>;
};

export async function GET(request: NextRequest, context: RouteContext) {
  return proxyTripServiceRequest(request, context);
}

export async function POST(request: NextRequest, context: RouteContext) {
  return proxyTripServiceRequest(request, context);
}

async function proxyTripServiceRequest(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  const targetUrl = new URL(path.join("/"), `${getTripServiceUrl()}/`);
  targetUrl.search = request.nextUrl.search;

  const headers = new Headers();
  copyHeader(request.headers, headers, "accept");
  copyHeader(request.headers, headers, "content-type");

  const hasBody = request.method !== "GET" && request.method !== "HEAD";
  const response = await fetch(targetUrl, {
    method: request.method,
    headers,
    body: hasBody ? await request.arrayBuffer() : undefined,
    cache: "no-store"
  });

  const responseHeaders = new Headers(response.headers);
  responseHeaders.delete("content-encoding");
  responseHeaders.delete("content-length");

  return new NextResponse(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: responseHeaders
  });
}

function copyHeader(source: Headers, target: Headers, name: string) {
  const value = source.get(name);
  if (value) {
    target.set(name, value);
  }
}

import { NextRequest, NextResponse } from "next/server";
import { getWorkerServiceInternalUrl } from "@/lib/config";

type RouteContext = {
  params: Promise<{
    path: string[];
  }>;
};

export async function GET(request: NextRequest, context: RouteContext) {
  return proxyWorkerServiceRequest(request, context);
}

export async function POST(request: NextRequest, context: RouteContext) {
  return proxyWorkerServiceRequest(request, context);
}

async function proxyWorkerServiceRequest(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  let targetUrl: URL;
  try {
    targetUrl = new URL(path.join("/"), `${getWorkerServiceInternalUrl()}/`);
  } catch (error) {
    return NextResponse.json(
      {
        error:
          error instanceof Error ? error.message : "Worker Service URL is not configured."
      },
      { status: 500 }
    );
  }
  targetUrl.search = request.nextUrl.search;

  const headers = new Headers();
  copyHeader(request.headers, headers, "accept");
  copyHeader(request.headers, headers, "authorization");
  copyHeader(request.headers, headers, "content-type");

  let response: Response;
  try {
    response = await fetch(targetUrl, {
      method: request.method,
      headers,
      body: request.method === "GET" ? undefined : await request.arrayBuffer(),
      cache: "no-store"
    });
  } catch {
    return NextResponse.json(
      {
        error: "Worker Service is unavailable. Confirm it is running and reachable from the web app."
      },
      { status: 503 }
    );
  }

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

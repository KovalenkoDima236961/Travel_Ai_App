import { NextRequest, NextResponse } from "next/server";
import { getNotificationServiceInternalUrl } from "@/lib/config";

type RouteContext = {
  params: Promise<{
    path: string[];
  }>;
};

export async function GET(request: NextRequest, context: RouteContext) {
  return proxyNotificationServiceRequest(request, context);
}

export async function PATCH(request: NextRequest, context: RouteContext) {
  return proxyNotificationServiceRequest(request, context);
}

// Only GET and PATCH are proxied: the user-facing Notification Service API is
// read + mark-read only. The internal batch-create endpoint is intentionally not
// reachable from the browser.
async function proxyNotificationServiceRequest(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;

  // Never proxy internal endpoints from the browser, even if requested.
  if (path[0] === "internal") {
    return NextResponse.json({ error: "Not found." }, { status: 404 });
  }

  let targetUrl: URL;
  try {
    targetUrl = new URL(path.join("/"), `${getNotificationServiceInternalUrl()}/`);
  } catch (error) {
    return NextResponse.json(
      {
        error:
          error instanceof Error
            ? error.message
            : "Notification Service URL is not configured."
      },
      { status: 500 }
    );
  }
  targetUrl.search = request.nextUrl.search;

  const headers = new Headers();
  copyHeader(request.headers, headers, "accept");
  copyHeader(request.headers, headers, "authorization");
  copyHeader(request.headers, headers, "content-type");

  const hasBody = request.method !== "GET" && request.method !== "HEAD";
  let response: Response;
  try {
    response = await fetch(targetUrl, {
      method: request.method,
      headers,
      body: hasBody ? await request.arrayBuffer() : undefined,
      cache: "no-store"
    });
  } catch {
    return NextResponse.json(
      {
        error:
          "Notification Service is unavailable. Confirm it is running and reachable from the web app."
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

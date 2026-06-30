import { describe, expect, it } from "vitest";

import { parseSSEBuffer, parseSSEChunk } from "@/lib/notifications/sse-parser";

describe("parseSSEBuffer", () => {
  it("parses a single event", () => {
    const result = parseSSEBuffer('event: notification.created\ndata: {"ok":true}\n\n');

    expect(result.remainder).toBe("");
    expect(result.events).toHaveLength(1);
    expect(result.events[0]).toMatchObject({
      event: "notification.created",
      data: { ok: true },
      malformed: false
    });
  });

  it("parses multiple events in one buffer", () => {
    const result = parseSSEBuffer(
      'event: heartbeat\ndata: {"ts":"now"}\n\nevent: notification.created\ndata: {"id":"n1"}\n\n'
    );

    expect(result.events.map((event) => event.event)).toEqual([
      "heartbeat",
      "notification.created"
    ]);
  });

  it("keeps a split event boundary as remainder", () => {
    const first = parseSSEChunk("", 'event: notification.created\ndata: {"id"');
    expect(first.events).toHaveLength(0);
    expect(first.remainder).toContain("notification.created");

    const second = parseSSEChunk(first.remainder, ':"n1"}\n\n');
    expect(second.remainder).toBe("");
    expect(second.events[0]).toMatchObject({
      event: "notification.created",
      data: { id: "n1" }
    });
  });

  it("parses heartbeat events without special casing", () => {
    const result = parseSSEBuffer('event: heartbeat\ndata: {"ts":"2026-06-25T12:00:00Z"}\n\n');
    expect(result.events[0]).toMatchObject({
      event: "heartbeat",
      data: { ts: "2026-06-25T12:00:00Z" }
    });
  });

  it("parses presence snapshot events", () => {
    const result = parseSSEBuffer(
      'event: presence.snapshot\ndata: {"tripId":"t1","users":[{"userId":"u1","role":"owner","state":"viewing","connectedAt":"2026-06-25T12:00:00Z","lastSeenAt":"2026-06-25T12:00:00Z"}]}\n\n'
    );

    expect(result.events[0]).toMatchObject({
      event: "presence.snapshot",
      data: {
        tripId: "t1",
        users: [
          expect.objectContaining({
            userId: "u1",
            role: "owner",
            state: "viewing"
          })
        ]
      }
    });
  });

  it("parses split presence chunks", () => {
    const first = parseSSEChunk("", 'event: presence.snapshot\ndata: {"tripId":"t1","users"');
    expect(first.events).toHaveLength(0);

    const second = parseSSEChunk(first.remainder, ':[]}\n\n');
    expect(second.events[0]).toMatchObject({
      event: "presence.snapshot",
      data: { tripId: "t1", users: [] }
    });
  });

  it("parses activity created events", () => {
    const result = parseSSEBuffer(
      'event: activity.created\ndata: {"event":{"id":"e1","tripId":"t1","actorUserId":"u1","eventType":"comment_created","entityType":"comment","entityId":"c1","metadata":{},"createdAt":"2026-06-30T12:00:00Z"}}\n\n'
    );

    expect(result.events[0]).toMatchObject({
      event: "activity.created",
      data: {
        event: expect.objectContaining({
          id: "e1",
          tripId: "t1",
          eventType: "comment_created"
        })
      }
    });
  });

  it("parses split activity chunks", () => {
    const first = parseSSEChunk("", 'event: activity.created\ndata: {"event":{"id":"e1"');
    expect(first.events).toHaveLength(0);

    const second = parseSSEChunk(first.remainder, ',"tripId":"t1","metadata":{}}}\n\n');
    expect(second.events[0]).toMatchObject({
      event: "activity.created",
      data: { event: expect.objectContaining({ id: "e1", tripId: "t1" }) }
    });
  });

  it("keeps unknown events for the caller to ignore", () => {
    const result = parseSSEBuffer('event: something.new\ndata: {"ok":true}\n\n');
    expect(result.events[0]).toMatchObject({
      event: "something.new",
      data: { ok: true }
    });
  });

  it("handles malformed JSON gracefully", () => {
    const result = parseSSEBuffer("event: notification.created\ndata: {not-json}\n\n");
    expect(result.events[0]).toMatchObject({
      event: "notification.created",
      data: null,
      malformed: true
    });
  });
});

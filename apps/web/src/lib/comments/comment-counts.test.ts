import { describe, expect, it } from "vitest";

import { buildCommentCountMap, makeCommentItemKey } from "@/lib/comments/comment-counts";
import type { CommentCount } from "@/types/comments";

describe("makeCommentItemKey", () => {
  it("builds the stable day/item key", () => {
    expect(makeCommentItemKey(2, 3)).toBe("day-2-item-3");
    expect(makeCommentItemKey(1, 0)).toBe("day-1-item-0");
  });
});

describe("buildCommentCountMap", () => {
  it("maps grouped counts by item key", () => {
    const counts: CommentCount[] = [
      { dayNumber: 1, itemIndex: 0, count: 2 },
      { dayNumber: 2, itemIndex: 3, count: 5 }
    ];

    const map = buildCommentCountMap(counts);

    expect(map[makeCommentItemKey(1, 0)]).toBe(2);
    expect(map[makeCommentItemKey(2, 3)]).toBe(5);
    expect(map[makeCommentItemKey(9, 9)]).toBeUndefined();
  });

  it("returns an empty map for no counts", () => {
    expect(buildCommentCountMap([])).toEqual({});
  });
});

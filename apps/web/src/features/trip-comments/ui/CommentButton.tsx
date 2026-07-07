"use client";

import { Button } from "@/shared/ui/button";

type CommentButtonProps = {
  count: number;
  onClick: () => void;
  disabled?: boolean;
};

export function CommentButton({ count, onClick, disabled = false }: CommentButtonProps) {
  const label = count > 0 ? `Comments (${count})` : "Comments";
  return (
    <Button
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      size="sm"
      type="button"
      variant="ghost"
    >
      <svg
        aria-hidden="true"
        className="mr-1.5 h-4 w-4"
        fill="none"
        stroke="currentColor"
        strokeWidth={1.8}
        viewBox="0 0 24 24"
      >
        <path
          d="M7.5 8.25h9m-9 3.75h6m4.5 4.5-3.75-3.75H6A2.25 2.25 0 0 1 3.75 10.5V6A2.25 2.25 0 0 1 6 3.75h12A2.25 2.25 0 0 1 20.25 6v10.5Z"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
      Comments
      {count > 0 ? (
        <span className="ml-1.5 inline-flex min-w-5 items-center justify-center rounded-full bg-primary-100 px-1.5 text-xs font-semibold text-primary-700">
          {count}
        </span>
      ) : null}
    </Button>
  );
}

export function AlphaReleaseNotice() {
  const label = process.env.NEXT_PUBLIC_ALPHA_RELEASE_LABEL?.trim();
  if (!label) return null;

  const supportHref = process.env.ALPHA_SUPPORT_URL?.trim() || "/settings";
  const feedbackHref = process.env.ALPHA_FEEDBACK_URL?.trim() || supportHref;

  return (
    <div
      className="border-b border-sand-200 bg-[#FFFDFA] px-4 py-2 text-[12px] text-cocoa-600"
      role="status"
    >
      <div className="mx-auto flex max-w-[1360px] flex-wrap items-center gap-x-3 gap-y-1">
        <span className="rounded-full border border-[#E5C3B6] bg-[#FBF0EB] px-2 py-0.5 font-semibold uppercase text-[#B3402E]">
          {label}
        </span>
        <a className="font-semibold text-clay hover:text-cocoa-900" href={feedbackHref}>
          Feedback
        </a>
        <a className="font-semibold text-clay hover:text-cocoa-900" href={supportHref}>
          Support
        </a>
      </div>
    </div>
  );
}

"use client";

import { useState } from "react";
import type { NotificationDigest } from "@/entities/notification/model";
import { formatRelativeTime } from "@/lib/notifications/relative-time";

export function NotificationDigestCard({ digest }: { digest: NotificationDigest }) {
  const [expanded,setExpanded]=useState(false);
  return <article className="rounded-[18px] border border-[#D9CDBB] bg-[#FFF9EF] p-5">
    <button type="button" className="flex w-full items-start justify-between gap-4 text-left" aria-expanded={expanded} onClick={()=>setExpanded((value)=>!value)}><span><span className="block text-[14px] font-semibold text-cocoa-900">{digest.eventCount} updates in your {digest.mode.replace("_digest","")} digest</span><span className="mt-1 block text-[12.5px] text-cocoa-400">{digest.channel.replace("_","-")} · scheduled {formatRelativeTime(digest.scheduledFor)}</span></span><span className="text-[12px] font-semibold text-clay-deep">{expanded?"Hide":"Preview"}</span></button>
    {expanded?<ul className="mt-4 space-y-2 border-t border-sand-300 pt-4">{digestPreviewLines(digest).map((line)=><li key={line.id} className="text-[12.5px] text-cocoa-600"><span className="font-semibold text-cocoa-800">{line.category}:</span> {line.message}</li>)}</ul>:null}
  </article>;
}

export function digestPreviewLines(digest: NotificationDigest) {
  return digest.items.map((item)=>({id:item.id,category:item.category.replaceAll("_"," "),message:`${item.message}${item.eventCount>1?` (${item.eventCount} updates)`:""}`}));
}

"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { deleteTripNotificationMute, getTripNotificationMutes, notificationPreferenceKeys, upsertTripNotificationMute } from "@/lib/api/notification-preferences";
import type { NotificationCategory } from "@/entities/notification-preferences/model";
import { cn } from "@/shared/lib/cn";

const choices: Array<{ category: NotificationCategory | null; label: string }> = [
  { category: null, label: "Mute trip" }, { category: "comments", label: "Comments" },
  { category: "trip_updates", label: "Trip updates" }, { category: "checklist", label: "Checklist" },
  { category: "reminders", label: "Reminders" }, { category: "expenses", label: "Expenses" }
];

export function TripMuteSettings({ tripId, className }: { tripId: string; className?: string }) {
  const client=useQueryClient(); const [mutedUntil,setMutedUntil]=useState("");
  const query=useQuery({queryKey:notificationPreferenceKeys.tripMutes(tripId),queryFn:()=>getTripNotificationMutes(tripId),enabled:Boolean(tripId)});
  const refresh=()=>client.invalidateQueries({queryKey:notificationPreferenceKeys.tripMutes(tripId)});
  const upsert=useMutation({mutationFn:(category:NotificationCategory|null)=>upsertTripNotificationMute({tripId,category,mutedUntil:mutedUntil?new Date(mutedUntil).toISOString():null}),onSuccess:refresh});
  const remove=useMutation({mutationFn:deleteTripNotificationMute,onSuccess:refresh}); const items=query.data??[];
  const active=(category:NotificationCategory|null)=>items.find((item)=>(item.category??null)===category);
  return <section className={cn("rounded-[16px] border border-sand-300 bg-white px-5 py-4",className)}>
    <div className="flex flex-wrap items-start justify-between gap-3"><div><h2 className="text-[14px] font-semibold text-cocoa-900">Notifications for this trip</h2><p className="mt-1 text-[12.5px] text-cocoa-400">Mute routine updates. Security and critical assigned actions still arrive.</p></div><label className="text-[12px] font-medium text-cocoa-500">Mute until<input aria-label="Mute until" className="ml-2 h-9 rounded-lg border border-sand-400 px-2 text-[12px]" type="datetime-local" value={mutedUntil} onChange={(event)=>setMutedUntil(event.target.value)}/></label></div>
    <div className="mt-3 flex flex-wrap gap-2">{choices.map((choice)=>{const mute=active(choice.category);return <button key={choice.label} type="button" disabled={query.isPending||upsert.isPending||remove.isPending} onClick={()=>mute?remove.mutate(mute.id):upsert.mutate(choice.category)} className={cn("rounded-full border px-3 py-1.5 text-[12.5px] font-medium transition",mute?"border-clay bg-clay-tint text-clay-deep":"border-sand-400 bg-white text-cocoa-600 hover:border-sand-600")}>{mute?`Unmute ${choice.label.toLowerCase()}`:choice.label}</button>})}</div>
    {query.isError||upsert.isError||remove.isError?<p className="mt-3 text-[12px] text-clay-deep" role="alert">Could not update trip notification settings.</p>:null}
  </section>;
}

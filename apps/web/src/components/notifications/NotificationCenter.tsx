"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { AppNotification } from "@/entities/notification/model";
import { listPendingNotificationDigests, notificationKeys } from "@/lib/api/notifications";
import { NotificationDigestCard } from "./NotificationDigestCard";
import { NotificationFilters } from "./NotificationFilters";
import { NotificationGroupCard } from "./NotificationGroupCard";

export function NotificationCenter({ items, onSelect, onMarkTripRead, onMuteTrip, onMuteCategory }: { items: AppNotification[]; onSelect:(item:AppNotification)=>void; onMarkTripRead?:(tripId:string)=>void; onMuteTrip?:(tripId:string)=>void; onMuteCategory?:(tripId:string,category:string)=>void }) {
  const [category,setCategory]=useState("all"); const digests=useQuery({queryKey:notificationKeys.pendingDigests,queryFn:listPendingNotificationDigests});
  const categories=useMemo(()=>Array.from(new Set(items.map((item)=>item.category))).sort(),[items]);
  const groups=useMemo(()=>groupNotifications(category==="all"?items:items.filter((item)=>item.category===category)),[items,category]);
  return <div className="mt-6 space-y-6">{digests.data?.length?<section><h2 className="mb-3 text-[12px] font-semibold uppercase tracking-[0.12em] text-cocoa-400">Pending digests</h2><div className="space-y-3">{digests.data.map((digest)=><NotificationDigestCard key={digest.id} digest={digest}/>)}</div></section>:null}{items.length?<div className="flex justify-end"><NotificationFilters categories={categories} value={category} onChange={setCategory}/></div>:null}{groups.map((dateGroup)=><section key={dateGroup.label}><h2 className="mb-3 text-[12px] font-semibold uppercase tracking-[0.12em] text-cocoa-400">{dateGroup.label}</h2><div className="space-y-3">{dateGroup.groups.map((group)=><NotificationGroupCard key={group.key} title={group.title} items={group.items} onSelect={onSelect} onMarkTripRead={onMarkTripRead} onMuteTrip={onMuteTrip} onMuteCategory={onMuteCategory}/>)}</div></section>)}</div>;
}

export function groupNotifications(items:AppNotification[], now = new Date()) {
  const dates=new Map<string,Map<string,AppNotification[]>>();
  for(const item of items){const dateLabel=dateBucket(item.createdAt,now);const trip=typeof item.metadata?.tripName==="string"?item.metadata.tripName:item.tripId?`Trip ${item.tripId.slice(0,8)}`:"Other updates";const key=`${trip}:${item.category}`;if(!dates.has(dateLabel))dates.set(dateLabel,new Map());const groups=dates.get(dateLabel)!;groups.set(key,[...(groups.get(key)??[]),item])}
  return Array.from(dates.entries()).map(([label,groups])=>({label,groups:Array.from(groups.entries()).map(([key,groupItems])=>({key,title:`${typeof groupItems[0]?.metadata?.tripName==="string"?groupItems[0].metadata.tripName:groupItems[0]?.tripId?`Trip ${groupItems[0].tripId.slice(0,8)}`:"Other"} · ${groupItems[0]?.category.replaceAll("_"," ")}`,items:groupItems}))}));
}
export function dateBucket(value:string,now=new Date()){const date=new Date(value);const start=new Date(now.getFullYear(),now.getMonth(),now.getDate()).getTime();const day=new Date(date.getFullYear(),date.getMonth(),date.getDate()).getTime();const diff=Math.round((start-day)/86400000);if(diff<=0)return"Today";if(diff===1)return"Yesterday";if(diff<7)return"This week";return"Older"}

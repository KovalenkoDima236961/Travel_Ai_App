"use client";

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CategoryDeliveryModeSelect } from "./CategoryDeliveryModeSelect";
import { NotificationDigestSettings } from "./NotificationDigestSettings";
import { QuietHoursSettings } from "./QuietHoursSettings";
import { PrimaryButton, SaveNotice, SectionHeading, SettingsCard } from "@/components/settings/controls";
import { getNotificationPreferences, notificationPreferenceKeys, updateNotificationPreferences } from "@/lib/api/notification-preferences";
import { getErrorMessage } from "@/lib/utils";
import type { NotificationCategory, NotificationChannel, NotificationDeliveryMode, NotificationPreference, NotificationSettings } from "@/entities/notification-preferences/model";

const channels: Array<{ value: NotificationChannel; label: string }> = [{value:"in_app",label:"In-app"},{value:"email",label:"Email"},{value:"push",label:"Push"}];
const categories: Array<{ value: NotificationCategory; label: string; description: string }> = [
  {value:"collaboration",label:"Collaboration",description:"Invites, availability, and group decisions."},
  {value:"comments",label:"Comments",description:"New itinerary comments."},
  {value:"role_changes",label:"Role changes",description:"Trip access and collaborator role changes."},
  {value:"trip_updates",label:"Trip updates",description:"Itinerary and route changes."},
  {value:"checklist",label:"Checklist",description:"Assignments, progress, and overdue items."},
  {value:"checklist_reminders",label:"Checklist reminders",description:"Assigned checklist reminders and nudges."},
  {value:"reminders",label:"Reminders",description:"Due preparation reminders."},
  {value:"pre_trip_reminders",label:"Pre-trip reminders",description:"Time-sensitive reminders before departure."},
  {value:"expenses",label:"Expenses",description:"New shared expenses."},
  {value:"settlements",label:"Settlements",description:"Pending, paid, and overdue settlements."},
  {value:"approval",label:"Approvals",description:"Approval requests and changes."},
  {value:"budget",label:"Budget",description:"Budget confidence and limit changes."},
  {value:"health",label:"Trip Health",description:"Planning risks and critical issues."},
  {value:"offline_sync",label:"Offline sync",description:"Sync conflicts and failures."},
  {value:"calendar",label:"Calendar",description:"Calendar sync status."},
  {value:"ai_generation",label:"AI generation",description:"Generation completion and failures."},
  {value:"security",label:"Security",description:"Share and access security changes."},
  {value:"system",label:"System",description:"Service and account-level notification updates."}
];
const fallbackSettings: NotificationSettings = {quietHoursEnabled:false,quietHoursStart:"22:00",quietHoursEnd:"08:00",quietHoursTimezone:Intl.DateTimeFormat().resolvedOptions().timeZone||"UTC",urgentBypassesQuietHours:true,dailyDigestTime:"08:00",weeklyDigestDay:1,weeklyDigestTime:"08:00"};

export function NotificationPreferencesV2() {
  const queryClient=useQueryClient(); const query=useQuery({queryKey:notificationPreferenceKeys.all,queryFn:getNotificationPreferences});
  const [items,setItems]=useState<NotificationPreference[]>([]); const [settings,setSettings]=useState(fallbackSettings);
  useEffect(()=>{if(query.data){setItems(query.data.items);setSettings(query.data.settings??fallbackSettings)}},[query.data]);
  const mutation=useMutation({mutationFn:updateNotificationPreferences,onSuccess:async(data)=>{setItems(data.items);setSettings(data.settings);queryClient.setQueryData(notificationPreferenceKeys.all,data);await queryClient.invalidateQueries({queryKey:notificationPreferenceKeys.all})}});
  const byKey=useMemo(()=>new Map(items.map((item)=>[`${item.channel}:${item.category}`,item])),[items]);
  const mode=(channel:NotificationChannel,category:NotificationCategory):NotificationDeliveryMode=>byKey.get(`${channel}:${category}`)?.deliveryMode??"muted";
  const setMode=(channel:NotificationChannel,category:NotificationCategory,deliveryMode:NotificationDeliveryMode)=>setItems((current)=>{const key=`${channel}:${category}`;let found=false;const next=current.map((item)=>{if(`${item.channel}:${item.category}`!==key)return item;found=true;return{...item,deliveryMode,enabled:deliveryMode!=="muted"}});if(!found)next.push({channel,category,deliveryMode,enabled:deliveryMode!=="muted"});return next});
  const error=query.error?getErrorMessage(query.error,"Could not load notification preferences."):mutation.error?getErrorMessage(mutation.error,"Could not save notification preferences."):null;
  return <SettingsCard><SectionHeading title="Notification preferences" subtitle="Choose instant delivery, a grouped digest, or mute by channel and category." />
    {query.isPending?<div className="mt-6 h-48 animate-pulse rounded-2xl bg-sand-200"/>:null}{error?<div className="mt-5"><SaveNotice errorMessage={error}/></div>:null}
    {!query.isPending&&!query.error?<form className="mt-6 space-y-5" onSubmit={(event)=>{event.preventDefault();mutation.mutate({items,settings})}}>
      <div className="overflow-x-auto rounded-2xl border border-sand-300"><table className="min-w-[760px] w-full text-left"><thead className="bg-sand-100"><tr><th className="px-4 py-3 text-[13px] font-semibold text-cocoa-700">Category</th>{channels.map((channel)=><th key={channel.value} className="px-3 py-3 text-[13px] font-semibold text-cocoa-700">{channel.label}</th>)}</tr></thead><tbody>{categories.map((category,index)=><tr key={category.value} className={index?"border-t border-sand-200":""}><td className="px-4 py-3"><p className="text-[13.5px] font-semibold text-cocoa-900">{category.label}</p><p className="text-[12px] text-cocoa-400">{category.description}</p></td>{channels.map((channel)=><td key={channel.value} className="px-3 py-3"><CategoryDeliveryModeSelect label={`${category.label} ${channel.label} delivery`} disabled={mutation.isPending} value={mode(channel.value,category.value)} onChange={(value)=>setMode(channel.value,category.value,value)}/></td>)}</tr>)}</tbody></table></div>
      <NotificationDigestSettings value={settings} onChange={setSettings} disabled={mutation.isPending}/><QuietHoursSettings value={settings} onChange={setSettings} disabled={mutation.isPending}/>
      <p className="text-[12.5px] text-cocoa-400">Security-critical notifications may remain instant. Trip mutes do not suppress security, assigned due reminders, offline conflicts, or critical Trip Health alerts.</p>
      {mutation.isSuccess?<SaveNotice successMessage="Notification preferences saved."/>:null}<div className="flex justify-end"><PrimaryButton type="submit" disabled={mutation.isPending}>{mutation.isPending?"Saving…":"Save preferences"}</PrimaryButton></div>
    </form>:null}
  </SettingsCard>;
}

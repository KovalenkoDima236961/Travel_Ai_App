"use client";

export function NotificationFilters({ categories, value, onChange }: { categories: string[]; value: string; onChange: (value:string)=>void }) {
  return <label className="text-[12px] font-semibold text-cocoa-500">Category<select className="ml-2 h-9 rounded-full border border-sand-400 bg-white px-3 text-[12.5px] text-cocoa-700" value={value} onChange={(event)=>onChange(event.target.value)}><option value="all">All</option>{categories.map((category)=><option key={category} value={category}>{category.replaceAll("_"," ")}</option>)}</select></label>;
}

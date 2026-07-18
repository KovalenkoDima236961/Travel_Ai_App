import Link from "next/link";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { AccountMenu } from "@/components/layout/AccountMenu";
import { GlobeIcon, PlusIcon } from "@/_pages/trips/ui/icons";

const navBase = "inline-flex h-[38px] items-center rounded-full px-4 text-[14.5px] transition";
const idle = "font-medium text-cocoa-500 hover:bg-sand-200 hover:text-cocoa-900";

export function TripLibraryHeader() { return <header className="sticky top-0 z-40 border-b border-sand-300 bg-sand-50/95 backdrop-blur"><div className="mx-auto flex max-w-[1280px] items-center justify-between gap-6 px-6 py-3 sm:px-10"><div className="flex items-center gap-6 lg:gap-9"><Link href="/" className="flex items-center gap-2.5 text-cocoa-900"><span className="flex h-8 w-8 items-center justify-center rounded-full bg-clay text-sand-100"><GlobeIcon className="h-[18px] w-[18px]" /></span><span className="font-newsreader text-[19px] font-semibold">Travel AI Planner</span></Link><nav className="hidden items-center gap-1 md:flex"><Link href="/trips" className={`${navBase} ${idle}`}>Trips</Link><Link href="/library" aria-current="page" className={`${navBase} bg-sand-200 font-semibold text-cocoa-900`}>Library</Link><Link href="/templates" className={`${navBase} ${idle}`}>Templates</Link><Link href="/workspaces" className={`${navBase} ${idle}`}>Workspaces</Link></nav></div><div className="flex items-center gap-3"><NotificationBell /><Link href="/trips/new" className="hidden h-[38px] items-center gap-2 rounded-full bg-clay px-[18px] text-[14px] font-semibold text-sand-100 sm:inline-flex"><PlusIcon className="h-[15px] w-[15px]"/>New trip</Link><AccountMenu /></div></div></header>; }

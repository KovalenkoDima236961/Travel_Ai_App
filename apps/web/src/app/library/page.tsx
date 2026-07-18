import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TripLibraryPage } from "@/components/library/TripLibraryPage";

export default function LibraryPage() { return <ProtectedRoute><TripLibraryPage /></ProtectedRoute>; }

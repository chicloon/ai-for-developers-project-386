import ProtectedRoute from "@/components/auth/ProtectedRoute";
import AppShell from "@/components/navigation/AppShell";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <ProtectedRoute>
      <AppShell>{children}</AppShell>
    </ProtectedRoute>
  );
}

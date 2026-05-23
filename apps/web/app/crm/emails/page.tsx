import { redirect } from "next/navigation";

export default function LegacyCRMEmailsRedirect({ searchParams }: { searchParams?: Record<string, string | string[] | undefined> }) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(searchParams ?? {})) {
    if (Array.isArray(value)) {
      for (const item of value) params.append(key, item);
    } else if (typeof value === "string") {
      params.set(key, value);
    }
  }
  redirect(`/somis-intl/crm/emails${params.toString() ? `?${params.toString()}` : ""}`);
}

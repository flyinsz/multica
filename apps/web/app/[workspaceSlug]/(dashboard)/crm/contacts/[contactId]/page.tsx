"use client";

import { useParams } from "next/navigation";
import { CRMContactDetailPage } from "@multica/views/crm/components";

export default function Page() {
  const params = useParams<{ contactId: string }>();
  return <CRMContactDetailPage contactId={params.contactId} />;
}

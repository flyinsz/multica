"use client";

import { useEffect } from "react";

export default function LegacyCRMEmailsRedirect() {
  useEffect(() => {
    window.location.replace(`/somis-intl/crm/emails${window.location.search || ""}`);
  }, []);

  return null;
}

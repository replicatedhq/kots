// This hook has not been integrated yet. It's considered a work in progress
import { useParams } from "react-router-dom";
import { useState, useEffect } from "react";
import { useApps } from "@features/App";
import { App } from "@types";

import { KotsParams } from "@types";

function useSelectedApp(): App | null {
  const { slug } = useParams<KotsParams>();
  const { data } = useApps();

  const { apps = [] } = data || {};

  const [selectedApp, setSelectedApp] = useState<App | null>(
    apps?.find((app: App) => app.slug === slug) || null
  );

  useEffect(() => {
    setSelectedApp(apps?.find((app: App) => app.slug === slug) || null);
  }, [apps, slug]);

  return selectedApp;
}

export { useSelectedApp };

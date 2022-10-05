// This hook has not been integrated yet. It's considered a work in progress
import { useParams } from "react-router-dom";
import { useState, useEffect } from "react";
import { useApps } from "@features/App";
import { App } from "@types";

function useCurrentApp() {
  let { slug } = useParams<{ slug: string }>();
  let { data, isFetched } = useApps();

  const { apps = [] } = data || {};

  const [currentApp, setCurrentApp] = useState<App | null>(null);

  useEffect(() => {
    if (apps && isFetched) {
      setCurrentApp(apps.find((app: App) => app.slug === slug) || null);
    }
  }, [apps, slug]);

  return { currentApp };
}

export { useCurrentApp };

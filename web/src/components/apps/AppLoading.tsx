import { useApps } from "@features/App";

import { Version } from "@types";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

const AppLoading = () => {
  const navigate = useNavigate();

  const { data: appsData } = useApps({ refetchInterval: 100000 });

  const { apps: appsList } = appsData || {};
  // Enforce initial app configuration (if exists)
  //   useEffect(() => {
  //     // Handle updating the theme state when switching apps.

  //     // Handle updating the theme state when switching apps.
  //     // Used for a fresh reload
  //     if (location.pathname === "/apps") {
  //       // updates state but does not cause infinite loop because app navigates away from /apps
  //       return;
  //     }

  //     // find if any app needs configuration and redirect to its configuration flow
  //     const appNeedsConfiguration = appsList?.find((app) => {
  //       return app?.downstream?.pendingVersions?.length > 0;
  //     });
  //     if (appNeedsConfiguration) {
  //       const downstream = appNeedsConfiguration.downstream;
  //       const firstVersion = downstream.pendingVersions.find(
  //         (version: Version) => version?.sequence === 0
  //       );
  //       if (firstVersion?.status === "pending_cluster_management") {
  //         navigate(`/${appNeedsConfiguration.slug}/cluster/manage`);
  //         return;
  //       }
  //       if (firstVersion?.status === "pending_config") {
  //         navigate(`/${appNeedsConfiguration.slug}/config`);
  //         return;
  //       }
  //     }
  //   }, [appsList]);
  return (
    <div className="flex justifyContent--center">
      <h1>Waiting for license to install...</h1>
    </div>
  );
};

export default AppLoading;

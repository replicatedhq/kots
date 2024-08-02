import { useApps } from "@features/App";

import { Version } from "@types";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

const AppLoading = () => {
  const navigate = useNavigate();

  const { data: appsData } = useApps({ refetchInterval: 100000 });

  const { apps: appsList } = appsData || {};

  useEffect(() => {
    const appNeedsConfiguration = appsList?.find((app) => {
      return app?.downstream?.pendingVersions?.length > 0;
    });
    if (appNeedsConfiguration) {
      const downstream = appNeedsConfiguration.downstream;
      const firstVersion = downstream.pendingVersions.find(
        (version: Version) => version?.sequence === 0
      );
      if (firstVersion?.status === "pending_cluster_management") {
        navigate(`/${appNeedsConfiguration.slug}/cluster/manage`);
        return;
      }
      if (firstVersion?.status === "pending_config") {
        navigate(`/${appNeedsConfiguration.slug}/config`);
        return;
      }
    }
  }, [appsList]);
  return (
    <div className="flex justifyContent--center alignItems--center tw-h-full">
      <p>Waiting for license to install...</p>
    </div>
  );
};

export default AppLoading;

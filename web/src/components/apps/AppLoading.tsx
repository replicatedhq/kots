import { useApps } from "@features/App";
import { Version } from '@types'
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Utilities } from "@src/utilities/utilities";

const AppLoading = ({ appSlugFromMetadata }) => {
  const navigate = useNavigate();

  const { data: appsData } = useApps({ refetchInterval: 3000 });

  const { apps: appsList } = appsData || {};

  useEffect(() => {

    const appNeedsConfiguration = appsList?.find((app) => {
      return app?.downstream?.pendingVersions?.length > 0;
    });

    // if no apps are present, return
    if (appsList?.length === 0) {
      return;
    }
    // if app needs configuration, redirect to the first app depending on the status
    if (Utilities.isInitialAppInstall(appNeedsConfiguration) && appNeedsConfiguration) {

      const downstream = appNeedsConfiguration.downstream;
      const firstVersion = downstream.pendingVersions.find(
        (version: Version) => version?.sequence === 0
      );
      if (
        firstVersion?.status === "pending_cluster_management"
      ) {
        navigate(`/${appNeedsConfiguration.slug}/cluster/manage`);
        return;
      }
      if (firstVersion?.status === "pending_config") {
        navigate(`/${appNeedsConfiguration.slug}/config`);
        return;
      }
      if (firstVersion?.status === "pending_preflight" || firstVersion?.status === "pending") {
        navigate(`/${appNeedsConfiguration.slug}/preflight`);
        return;
      }

    } else {
      // a version has already been deployed
      navigate(`/app/${appSlugFromMetadata}`);
    }

  }, [appsData]);
  return (
    <div className="flex justifyContent--center alignItems--center tw-h-full">
      <p>Waiting for license to install...</p>
    </div>
  );
};

export default AppLoading;


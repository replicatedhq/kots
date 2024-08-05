import { useApps } from "@features/App";

import { Version } from "@types";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

const AppLoading = ({ appSlugFromMetadata }) => {
  const navigate = useNavigate();

  const { data: appsData } = useApps({ refetchInterval: 3000 });

  const { apps: appsList } = appsData || {}
  
  useEffect(() => {
    const appNeedsConfiguration = appsList?.find((app) => {
      return app?.downstream?.pendingVersions?.length > 0;
    });
    if (appNeedsConfiguration) {
      const downstream = appNeedsConfiguration.downstream;
      const firstVersion = downstream.pendingVersions.find(
        (version: Version) => version?.sequence === 0
      );

      if (firstVersion) {
       navigate(`/${appNeedsConfiguration.slug}/cluster/manage`);
        return;
      }
   
    } else  {
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

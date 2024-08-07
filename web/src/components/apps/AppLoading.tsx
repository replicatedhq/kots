import { useApps } from "@features/App";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

const AppLoading = () => {
  const navigate = useNavigate();

  const { data: appsData } = useApps({ refetchInterval: 3000 });

  const { apps: appsList } = appsData || {};

  useEffect(() => {
    // if no apps are present, return
    if (appsList?.length === 0) {
      return;
    } else {
      navigate("/");
    }
  }, [appsData]);
  return (
    <div className="flex justifyContent--center alignItems--center tw-h-full">
      <p>Waiting for license to install...</p>
    </div>
  );
};

export default AppLoading;

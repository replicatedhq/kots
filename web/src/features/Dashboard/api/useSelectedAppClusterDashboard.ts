import { useRef, useState, useEffect } from "react";
import { useSelectedAppClusterDashboard } from "./getSelectedAppClusterDashboard";
import { Utilities } from "@src/utilities/utilities";
import { slowLoadingThreshold } from "@src/constants/timers";
import axios from "axios";

axios.defaults.withCredentials = true;

function useSelectedAppClusterDashboardWithIntercept(
  { refetchInterval }: { refetchInterval: number | false } = {
    refetchInterval: false,
  }
) {
  const [isSlowLoading, setIsSlowLoading] = useState(false);

  let timerId = useRef<null | NodeJS.Timeout>(null);

  let appClusterDashboardQuery = useSelectedAppClusterDashboard({
    refetchInterval,
  });

  useEffect(() => {
    axios.interceptors.request.use(
      (x) => {
        if (x.url?.endsWith("/dashboard")) {
          if (timerId.current) {
            clearTimeout(timerId.current);
            timerId.current = null;
          }
          timerId.current = setTimeout(
            () => setIsSlowLoading(true),
            slowLoadingThreshold
          );
          return x;
        }
        return x;
      },
      (error) => {
        if (error?.response?.status === 401) {
          Utilities.logoutUser();
          return;
        }
        //TODO: handle other errors
        console.log("Error => axios.interceptors.request", error);
      }
    );
    axios.interceptors.response.use(
      async (x) => {
        if (x?.config?.url?.endsWith("/dashboard")) {
          //await sleep();
          setIsSlowLoading(false);
          if (timerId.current) {
            clearTimeout(timerId.current);
            timerId.current = null;
          }

          return x;
        }
        return x;
      },
      (error) => {
        if (error?.response?.status === 401) {
          Utilities.logoutUser();
          return;
        }
        //TODO: handle other errors
        console.log("Error => axios.interceptors.request", error);
      }
    );

    return () => {
      if (timerId.current) {
        clearTimeout(timerId.current);
      }
    };
  }, []);
  // return everything that comes with react query
  return { ...appClusterDashboardQuery, isSlowLoading };
}

export { useSelectedAppClusterDashboardWithIntercept };

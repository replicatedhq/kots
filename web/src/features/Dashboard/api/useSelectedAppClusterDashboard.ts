import { useRef, useState, useEffect } from "react";
import { useSelectedAppClusterDashboard } from "./getSelectedAppClusterDashboard";
import { Utilities } from "@src/utilities/utilities";
import axios from "axios";

function useSelectedAppClusterDashboardWithIntercept({
  refetchInterval = false,
}: {
  refetchInterval: number | false;
}) {
  const [isSlowLoading, setIsSlowLoading] = useState(false);

  let timerId = useRef<null | NodeJS.Timeout>(null);

  // DEV: sleep function to delay the return of the data
  // use this to simulate a slow loading request
  // const sleep = (ms = 1000): Promise<void> => {
  //   return new Promise<void>((resolve) => {
  //     setTimeout(() => resolve(), ms);
  //   });
  // };

  let appClusterDashboardQuery =
    useSelectedAppClusterDashboard({ refetchInterval });

  useEffect(() => {
    axios.interceptors.request.use(
      (x) => {
        // not sure if this check is enough to make sure that it
        // only happens on /license endpoint
        if (x.url?.endsWith("/dashboard")) {
          // set timeout to 500ms, change it to whatever you want
          timerId.current = setTimeout(() => setIsSlowLoading(true), 500);
          return x;
        }
        return x;
      },
      (error) => {
        if (error.response.status === 401) {
          Utilities.logoutUser();
          return;
        }
        //TODO: handle other errors
        console.log("Error => axios.interceptors.request", error);
      }
    );
    axios.interceptors.response.use(
      async (x) => {
        if (x.config.url?.endsWith("/dashboard")) {
          //await sleep();
          setIsSlowLoading(false);
          if (timerId.current) {
            clearTimeout(timerId.current);
          }

          return x;
        }
        return x;
      },
      (error) => {
        if (error.response.status === 401) {
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

import { useRef, useState, useEffect } from "react";
import { useNextAppVersion } from "./getNextAppVersion";
import axios from "axios";

function useNextAppVersionWithIntercept(appSlug: string) {
  const [isSlowLoading, setIsSlowLoading] = useState(false);

  let timerId = useRef<null | NodeJS.Timeout>(null);

  // DEV: sleep function to delay the return of the data
  // use this to simulate a slow loading request
  // const sleep = (ms = 1000): Promise<void> => {
  //   return new Promise<void>((resolve) => {
  //     setTimeout(() => resolve(), ms);
  //   });
  // };

  let nextAppVersionQuery = useNextAppVersion(appSlug);

  useEffect(() => {
    axios.interceptors.request.use(
      (x) => {
        // not sure if this check is enough to make sure that it
        // only happens on /license endpoint
        if (x.url?.endsWith("/next-app-version")) {
          // set timeout to 500ms, change it to whatever you want
          timerId.current = setTimeout(() => setIsSlowLoading(true), 500);
          return x;
        }
        return x;
      },
      (error) => {
        if (error.response.status === 401) {
          // TODO: handle unauthorized
          console.log("an error occurred");
        }
        //TODO: handle other errors
        console.log("Error => axios.interceptors.request", error);
      }
    );
    axios.interceptors.response.use(
      async (x) => {
        if (x.config.url?.endsWith("/next-app-version")) {
          //await sleep();
          setIsSlowLoading(false);
          clearTimeout(timerId.current as NodeJS.Timeout);

          return x;
        }
        return x;
      },
      (error) => {
        if (error.response.status === 401) {
          // TODO: handle unauthorized
          console.log("an error occurred");
        }
        //TODO: handle other errors
        console.log("Error => axios.interceptors.request", error);
      }
    );
  }, []);
  // return everything that comes with react query
  return { ...nextAppVersionQuery, isSlowLoading };
}

export { useNextAppVersionWithIntercept };

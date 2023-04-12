import { useRef, useState, useEffect } from "react";
import { useLicense } from "./getLicense";
import axios from "axios";
import { slowLoadingThreshold } from "@src/constants/timers";

axios.defaults.withCredentials = true;

function useLicenseWithIntercept() {
  const [isSlowLoading, setIsSlowLoading] = useState(false);

  let timerId = useRef<null | NodeJS.Timeout>(null);

  let licenseQuery = useLicense();

  useEffect(() => {
    axios.interceptors.request.use(
      (x) => {
        if (x.url?.endsWith("/license")) {
          if (timerId.current) {
            clearTimeout(timerId.current);
            timerId.current = null;
          }
          // set timeout to 500ms, change it to whatever you want
          timerId.current = setTimeout(() => {
            setIsSlowLoading(true);
          }, slowLoadingThreshold);
          return x;
        }
        return x;
      },
      (error) => {
        if (error?.response?.status === 401) {
          // TODO: handle unauthorized
          console.log("an error occurred");
        }
        //TODO: handle other errors
        console.log("Error => axios.interceptors.request", error);
      }
    );
    axios.interceptors.response.use(
      async (x) => {
        if (x?.config?.url?.endsWith("/license")) {
          //await sleep();
          setIsSlowLoading(false);
          clearTimeout(timerId.current as NodeJS.Timeout);
          timerId.current = null;
          return x;
        }
        return x;
      },
      (error) => {
        if (error?.response?.status === 401) {
          // TODO: handle unauthorized
          console.log("an error occurred");
        }
        //TODO: handle other errors
        console.log("Error => axios.interceptors.request", error);
      }
    );
  }, []);
  // return everything that comes with react query
  return { ...licenseQuery, isSlowLoading };
}

export { useLicenseWithIntercept };

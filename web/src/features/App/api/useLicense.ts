import { useRef, useState, useEffect } from "react";
import { useLicense } from "./getLicense";
import axios from "axios";

function useLicenseWithIntercept(appSlug: string) {
  const [isLoading, setIsLoading] = useState(false);

  let timerId = useRef(null);
  let axiosId = null;

  const sleep = (ms = 1000): Promise<void> => {
    return new Promise<void>((resolve) => {
      setTimeout(() => resolve(), ms);
    });
  };
  let { data } = useLicense(appSlug);

  console.log("data in uselicense", data);
  useEffect(() => {
    axiosId = axios.interceptors.request.use(
      (x) => {
        // not sure if this is enough to make sure that it only happens on /license endpoint
        if (x.url?.endsWith("/license")) {
          // set timeout to 500ms, change it to whatever you want
          timerId.current = setTimeout(() => setIsLoading(true), 500);
          return x;
        }
        return x;
      },
      (error) => {
        if (error.response.status === 401) {
          console.log("do nothing");
        }
        console.log("Error => axios.interceptors.request");
        console.log(error);
      }
    );
    axios.interceptors.response.use(async (x) => {
      if (x.config.url?.endsWith("/license")) {
        await sleep();
        setIsLoading(false);
        clearTimeout(timerId.current);

        return x;
      }
      return x;
    });
  }, []);
  return { data, isLoading };
}

export { useLicenseWithIntercept };

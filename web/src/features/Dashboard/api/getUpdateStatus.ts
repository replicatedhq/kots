import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

export const getUpdateStatus = async (appSlug: string) => {
  try {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/app/${appSlug}/task/updatedownload`,
      {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      }
    );

    if (res.ok && res.status == 200) {
      const appResponse = await res.json();
      console.log(appResponse, "appResponse");
      // if response status is not running, and not bundle uploading, stop updatechecker polling
      // set the states
      // getAppLicense
      // if there is updateCallback -> updateCallback()
      //startFetchAppDownstreamJob()
      return appResponse;
    } else {
      console.log("something went wrong");

      throw new Error("could not getAppDownstream");
    }
  } catch (err) {
    console.log("err");
    if (err instanceof Error) {
      throw err;
    }
  }
};

export const useUpdateStatus = () => {
  const { selectedApp } = useSelectedApp();

  return useQuery({
    queryFn: () => getUpdateStatus(selectedApp?.slug || ""),
    queryKey: ["getUpdateStatus"],
    onError: (err: Error) => console.log(err),
    refetchInterval: (data) => (data?.status !== "running" ? false : 1000),
  });
};

export default { useUpdateStatus };

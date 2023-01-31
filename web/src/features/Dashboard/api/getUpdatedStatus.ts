import { useMutation, useQueryClient } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

export const getUpdatedStatus = async (appSlug: string) => {
  console.log("fetching updated status");
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

export const useUpdatedStatus = () => {
  const { selectedApp } = useSelectedApp();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => getUpdatedStatus(selectedApp?.slug || ""),
    onError: (err: Error) => console.log(err),
    onSuccess: () => queryClient.invalidateQueries("getAppDownstream"),
  });
};

export default { useUpdatedStatus };

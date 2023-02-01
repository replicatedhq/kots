import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

export const createSnapshot = async (
  option: string,
  appSlug: string | undefined
) => {
  let url =
    option === "full"
      ? `${process.env.API_ENDPOINT}/snapshot/backup`
      : `${process.env.API_ENDPOINT}/app/${appSlug}/snapshot/backup`;

  try {
    const res = await fetch(url, {
      method: "POST",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
    });

    const response = await res.json();
    if (res.ok && res.status == 200) {
      // if response status is not running, and not bundle uploading, stop updatechecker polling
      // set the states
      // getAppLicense
      // if there is updateCallback -> updateCallback()
      //startFetchAppDownstreamJob()
      return response;
    } else {
      throw new Error(response.error);
    }
  } catch (err) {
    console.log(err);
    if (err instanceof Error) {
      throw err;
    }
  }
};

export const useCreateSnapshot = (option: string) => {
  const { selectedApp } = useSelectedApp();
  return useQuery({
    queryFn: () => createSnapshot(option, selectedApp?.slug),
    queryKey: ["createSnapshot"],
    onError: (err: Error) => console.log(err),
    //refetchInterval: (data) => (data?.status !== "running" ? false : 1000),
  });
};

export default { useCreateSnapshot };

import { isAwaitingResults } from "@src/utilities/utilities";
import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

export const getAppDownstream = async (appSlug: string) => {
  try {
    const res = await fetch(`${process.env.API_ENDPOINT}/app/${appSlug}`, {
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "GET",
    });

    if (res.ok && res.status == 200) {
      const appResponse = await res.json();
      // if we're not waiting on any results then stop the polling
      return appResponse;
      // wait a couple of seconds to avoid any race condiditons with the update checker then refetch the app to ensure we have the latest everything
      // this is hacky and I hate it but it's just building up more evidence in my case for having the FE be able to listen to BE envents
      // if that was in place we would have no need for this becuase the latest version would just be pushed down.
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

export const useAppDownstream = () => {
  const { selectedApp } = useSelectedApp();
  return useQuery({
    queryFn: () => getAppDownstream(selectedApp?.slug || ""),
    queryKey: ["getAppDownstream"],
    onError: (err: Error) => console.log(err),
    refetchInterval: (data) =>
      data && !isAwaitingResults(data?.downstream?.pendingVersions)
        ? false
        : 1000,
  });
};

export default { useAppDownstream };

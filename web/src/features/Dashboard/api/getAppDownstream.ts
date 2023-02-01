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

export const useAppDownstream = () => {
  const { selectedApp } = useSelectedApp();
  return useQuery({
    queryFn: () => getAppDownstream(selectedApp?.slug || ""),
    queryKey: ["getAppDownstream"],
    onError: (err: Error) => console.log(err),
    refetchInterval: (downstream) =>
      downstream && !isAwaitingResults(downstream?.pendingVersions)
        ? false
        : 2000,
    select: (data) => data?.downstream,
  });
};

export default { useAppDownstream };

import { isAwaitingResults } from "@src/utilities/utilities";
import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";
import { Downstream } from "@types";

export const getAppDownstream = async (
  appSlug: string
): Promise<Downstream | null> => {
  const res = await fetch(`${process.env.API_ENDPOINT}/app/${appSlug}`, {
    headers: {
      Authorization: Utilities.getToken(),
      "Content-Type": "application/json",
    },
    method: "GET",
  });

  if (!res.ok && res.status !== 200) {
    throw new Error(
      `an error occurred while fetching downstream for ${appSlug}`
    );
  }

  const appResponse = await res.json();
  return appResponse.downstream;
};

export const useAppDownstream = () => {
  const selectedApp = useSelectedApp();
  return useQuery({
    queryFn: () => getAppDownstream(selectedApp?.slug || ""),
    queryKey: ["getAppDownstream"],
    onError: (err: Error) => console.log(err),
    refetchInterval: (downstream) =>
      downstream && !isAwaitingResults(downstream?.pendingVersions)
        ? false
        : 2000,
  });
};

export default { useAppDownstream };

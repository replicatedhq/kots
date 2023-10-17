import { isAwaitingResults } from "@src/utilities/utilities";
import { useQuery } from '@tanstack/react-query';
import { useSelectedApp } from "@features/App";
import { Downstream } from "@types";

export const getAppDownstream = async (
  appSlug: string
): Promise<Downstream | null> => {
  const res = await fetch(`${process.env.API_ENDPOINT}/app/${appSlug}`, {
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
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

export const useAppDownstream = (
  onSuccess: (data: Downstream) => void,
  onError: (data: { message: string }) => void
) => {
  const selectedApp = useSelectedApp();
  return useQuery({
    queryFn: () => getAppDownstream(selectedApp?.slug || ""),
    queryKey: ["getAppDownstream"],
    onError,
    onSuccess,
    refetchInterval: (downstream) =>
      downstream && !isAwaitingResults(downstream?.pendingVersions)
        ? false
        : 2000,
  });
};

export default { useAppDownstream };

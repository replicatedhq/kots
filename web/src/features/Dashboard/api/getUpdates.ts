import { useQuery } from "@tanstack/react-query";
import { useSelectedApp } from "@features/App";

interface UpdateResponse {
  availableUpdates: number;
  currentAppSequence: number;
  currentRelease: { sequence: number; version: string };
  availableReleases: { sequence: number; version: string };
}
export interface Updates {
  availableUpdates?: number;
  checkingForUpdates: boolean;
  checkingForUpdatesError?: boolean;
  checkingUpdateMessage?: string;
  noUpdatesAvailable: boolean;
}

// bad name, will fix later
export const getCheckForUpdates = async (
  appSlug: string
): Promise<UpdateResponse> => {
  let res = await fetch(
    `${process.env.API_ENDPOINT}/app/${appSlug}/updatecheck`,
    {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "include",
      method: "POST",
    }
  );

  // on the dashboard page it triggers getAppLicense here
  if (res.ok) {
    const response = await res.json();
    return response;
  } else {
    const text = await res.text();
    throw new Error(text);
  }
};

const makeUpdatesResponse = (response: UpdateResponse): Updates => {
  return {
    ...response,
    checkingForUpdates: response.availableUpdates === 0 ? false : true,
    noUpdatesAvailable: response.availableUpdates === 0 ? true : false,
  };
  // sets timeout to 3 seconds and set noUpdatesAvailable to false
};

// update name later
export const useCheckForUpdates = (
  onSuccess: (response: Updates) => void,
  onError: (err: Error) => void
) => {
  const selectedApp = useSelectedApp();
  return useQuery({
    queryFn: () => getCheckForUpdates(selectedApp?.slug || ""),
    queryKey: ["getCheckForUpdates"],
    enabled: false,
    select: (data) => makeUpdatesResponse(data),
    onSuccess,
    onError,
    refetchOnWindowFocus: false,
  });
};

export default { useCheckForUpdates };

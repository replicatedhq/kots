import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

interface UpdateResponse {
  availableUpdates: number;
  currentAppSequence: number;
  currentRelease: { sequence: number; version: string };
  availableReleases: { sequence: number; version: string };
}
interface Updates {
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
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
    }
  );

  const response = await res.json();
  // on the dashboard page it triggers getAppLicense here
  if (res.ok) {
    return response;
  } else {
    throw new Error(response.error);
  }
};

const makeUpdatesResponse = (response: UpdateResponse): Updates => {
  return {
    checkingForUpdates: response.availableUpdates === 0 ? false : true,
    noUpdatesAvailable: response.availableUpdates === 0 ? true : false,
  };
  // sets timeout to 3 seconds and set noUpdatesAvailable to false
};

// update name later
export const useCheckForUpdates = () => {
  const { selectedApp } = useSelectedApp();
  return useQuery({
    queryFn: () => getCheckForUpdates(selectedApp?.slug || ""),
    queryKey: ["getCheckForUpdates"],
    onError: (err: Error) => console.log(err),
    enabled: true,
    select: (data) => makeUpdatesResponse(data),
  });
};

export default { useCheckForUpdates };

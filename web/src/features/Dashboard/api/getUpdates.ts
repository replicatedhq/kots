import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

interface Updates {
  checkingForUpdates: boolean;
  checkingForUpdatesError?: boolean;
  checkingUpdateMessage?: string;
  noUpdatesAvailable: boolean;
}

// bad name, will fix later
export const getCheckForUpdates = async (appSlug: string): Promise<number> => {
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
  // matter okay or not it gets getAppLicense
  if (res.ok) {
    return response;
  } else {
    // return these errors
    // setState({
    //   checkingForUpdateError: true,
    //   checkingForUpdates: false,
    //   checkingUpdateMessage: text
    //     ? text
    //     : "There was an error checking for updates.",
    // });
    throw new Error(response.error);
  }
};

const makeUpdatesResponse = (response: any): Updates => {
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
    enabled: false,
    select: (data) => makeUpdatesResponse(data),
  });
};

export default { useCheckForUpdates };

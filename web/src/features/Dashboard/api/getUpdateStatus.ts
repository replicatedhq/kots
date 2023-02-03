import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

interface UpdateStatusResponse {
  currentMessage: string;
  status: string;
}
interface UpdateStatus {
  checkingForUpdateError: boolean;
  checkingForUpdates: boolean;
  checkingUpdateMessage: string;
  status: string;
}

const getUpdateStatus = async (
  appSlug: string
): Promise<UpdateStatusResponse> => {
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

  if (!res.ok) {
    throw new Error("Error getting update status");
  }
  const appResponse = await res.json();
  return appResponse;
};

const makeUpdateStatusResponse = (
  response: UpdateStatusResponse
): UpdateStatus => {
  return {
    checkingForUpdateError: response.status === "failed",
    checkingForUpdates: response.status !== "running",
    checkingUpdateMessage: response.currentMessage,
    status: response.status,
  };
};

export const useUpdateStatus = () => {
  const { selectedApp } = useSelectedApp();

  return useQuery({
    queryFn: () => getUpdateStatus(selectedApp?.slug || ""),
    queryKey: ["getUpdateStatus"],
    onError: (err: Error) => console.log(err),
    refetchInterval: (data) => (data?.status !== "running" ? false : 1000),
    select: (response: UpdateStatusResponse) =>
      makeUpdateStatusResponse(response),
  });
};

export default { useUpdateStatus };

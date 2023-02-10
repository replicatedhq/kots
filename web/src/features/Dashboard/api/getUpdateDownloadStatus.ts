import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

export interface UpdateStatusResponse {
  currentMessage: string;
  status: string;
}

const getUpdateDownloadStatus = async (
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
    throw new Error("failed to get rewrite status");
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

export const useUpdateDownloadStatus = () => {
  const selectedApp = useSelectedApp();

  return useQuery({
    queryFn: () => getUpdateDownloadStatus(selectedApp?.slug || ""),
    queryKey: ["getUpdateStatus"],
    onSuccess,
    onError,
    refetchInterval: (data) =>
      data?.status !== "running" && !isBundleUploading ? false : 1000,
  });
};

export default { useUpdateDownloadStatus };

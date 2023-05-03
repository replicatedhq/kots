import { useQuery } from "react-query";
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
        "Content-Type": "application/json",
      },
      credentials: "include",
      method: "GET",
    }
  );

  if (!res.ok) {
    throw new Error("failed to get rewrite status");
  }
  const appResponse = await res.json();

  return appResponse;
};

export const useUpdateDownloadStatus = (
  onSuccess: (data: UpdateStatusResponse) => void,
  onError: (error: Error) => void,
  isBundleUploading: boolean
) => {
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

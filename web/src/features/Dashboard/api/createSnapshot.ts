import { useMutation } from "@tanstack/react-query";
import { useSelectedApp } from "@features/App";

interface SnapshotResponse {
  success: boolean;
  error: string;
  kotsadmNamespace: string;
  kotsadmRequiresVeleroAccess: boolean;
  option?: "full" | "partial" | undefined;
}

export const createSnapshot = async (
  option: "full" | "partial",
  appSlug: string
): Promise<SnapshotResponse> => {
  let url =
    option === "full"
      ? `${process.env.API_ENDPOINT}/snapshot/backup`
      : `${process.env.API_ENDPOINT}/app/${appSlug}/snapshot/backup`;

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
  });

  const response = await res.json();
  if (!res.ok && res.status !== 200) {
    const error = new Error("could not create a snapshot");
    throw error;
  }

  return { ...response, option };
};

export const useCreateSnapshot = (
  onSuccess: (data: SnapshotResponse) => void,
  onError: (error: Error) => void
) => {
  const selectedApp = useSelectedApp();
  return useMutation({
    mutationFn: (option: "full" | "partial") =>
      createSnapshot(option, selectedApp?.slug || ""),
    onSuccess,
    onError,
  });
};

export default { useCreateSnapshot };

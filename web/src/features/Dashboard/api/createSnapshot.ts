import { useMutation } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

interface SnapshotResponse {
  success: boolean;
  error: string;
  kotsadmNamespace: string;
  kotsadmRequiresVeleroAccess: boolean;
}
interface Snapshot {
  startingSnapshot: boolean;
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
      Authorization: Utilities.getToken(),
      "Content-Type": "application/json",
    },
  });

  const response = await res.json();
  if (!res.ok && res.status !== 200) {
    throw new Error(response.error);
  }

  return response;
};

// const createSnapshotResponse = (response: SnapshotResponse): Snapshot => {
//   return {
//     startingSnapshot: response.kotsadmRequiresVeleroAccess ? false : true,
//   };
// };

export const useCreateSnapshot = (
  onSuccess: () => void,
  onError: () => void
) => {
  const { selectedApp } = useSelectedApp();
  return useMutation({
    mutationFn: (option: "full" | "partial") =>
      createSnapshot(option, selectedApp?.slug || ""),
    onSuccess,
    onError,
  });
};

export default { useCreateSnapshot };

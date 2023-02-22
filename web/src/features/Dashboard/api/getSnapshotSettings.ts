import { SnapshotSettings } from "@types";
import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";

const getSnapshotSettings = async () => {
  const res = await fetch(`${process.env.API_ENDPOINT}/snapshots/settings`, {
    headers: {
      Authorization: Utilities.getToken(),
      "Content-Type": "application/json",
    },
    method: "GET",
  });
  const response = await res.json();
  if (!res.ok && res.status !== 200) {
    const error = new Error("could not create a snapshot");

    throw error;
  }

  return response;
};

export const useSnapshotSettings = (
  onSuccess: (data: SnapshotSettings) => void,
  onError: (error: Error) => void
) => {
  return useQuery({
    queryFn: () => getSnapshotSettings(),
    queryKey: ["getSnapshotSettings"],
    onSuccess,
    onError,
  });
};

export default { useSnapshotSettings };

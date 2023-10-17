import { SnapshotSettings } from "@types";
import { useQuery } from '@tanstack/react-query';

const getSnapshotSettings = async () => {
  const res = await fetch(`${process.env.API_ENDPOINT}/snapshots/settings`, {
    headers: {
      "Content-Type": "application/json",
    },
    method: "GET",
    credentials: "include",
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

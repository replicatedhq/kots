import { useQuery } from '@tanstack/react-query';
import { useSelectedApp } from "@features/App";

export const getAirgapConfig = async (appSlug: string): Promise<number> => {
  const configUrl = `${process.env.API_ENDPOINT}/app/${appSlug}/airgap/config`;

  let simultaneousUploads = 3;

  let res = await fetch(configUrl, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
  });
  const response = await res.json();
  if (res.ok) {
    simultaneousUploads = response.simultaneousUploads;

    return simultaneousUploads;
  } else {
    throw new Error(response.error);
  }
};

export const useAirgapConfig = (
  onSuccess: (simultaneousUploads: Number) => void
) => {
  const selectedApp = useSelectedApp();
  return useQuery({
    queryFn: () => getAirgapConfig(selectedApp?.slug || ""),
    queryKey: ["getAirgapConfig"],
    onError: (err: Error) => console.log(err),
    enabled: false,
    onSuccess,
  });
};

export default { useAirgapConfig };

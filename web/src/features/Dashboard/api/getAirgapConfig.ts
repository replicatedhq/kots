import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";

export const getAirgapConfig = async (appSlug: string) => {
  const configUrl = `${process.env.API_ENDPOINT}/app/${appSlug}/airgap/config`;

  // let simultaneousUploads = 3;
  try {
    let res = await fetch(configUrl, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: Utilities.getToken(),
      },
    });
    if (res.ok) {
      const response = await res.json();
      // simultaneousUploads = response.simultaneousUploads;
      return response;
    }

    // may need it later
    // airgapUploader.current = new AirgapUploader(
    //     true,
    //     app.slug,
    //     onDropBundle,
    //     simultaneousUploads
    //   );
  } catch (err) {
    console.log(err);
    if (err instanceof Error) {
      throw err;
    }
  }
};

export const useAirgapConfig = () => {
  const { selectedApp } = useSelectedApp();
  return useQuery({
    queryFn: () => getAirgapConfig(selectedApp?.slug || ""),
    queryKey: ["getAirgapConfig"],
    onError: (err: Error) => console.log(err),
    //refetchInterval: (data) => data
  });
};

export default { useAirgapConfig };

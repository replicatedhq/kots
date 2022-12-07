import axios from "axios";

import { Utilities } from "@src/utilities/utilities";
import { useQuery } from "react-query";

const fetchSupportBundleCommand = async (watchSlug: string, origin: string) => {
  const config = {
    headers: {
      Authorization: Utilities.getToken(),
      "Content-Type": "application/json",
    },
  };

  try {
    const res = await axios.post(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${watchSlug}/supportbundleommand`,
      origin,
      config
    );

    if (res.status === 200) {
      return res.data;
    }
  } catch (err) {
    if (err instanceof Error) {
      throw err;
    }
  }
};

export const useSupportBundleCommand = (watchSlug: string, origin: string) => {
  return useQuery(["supportBundleCommand"], () =>
    fetchSupportBundleCommand(watchSlug, origin)
  );
};

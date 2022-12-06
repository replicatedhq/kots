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
  console.log(origin, "oriign");

  try {
    const res = await axios.post(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${watchSlug}/supportbundlecommand`,
      origin,
      config
    );

    if (res.status === 200) {
      console.log(res.data, "res");
      return res.data;
    } else {
      // TODO: more error handling
      console.log("something went wrong");
      throw new Error("something went wrong");
    }
  } catch (err) {
    console.log("err");
    if (err instanceof Error) {
      throw err;
    }
  }
};

export const useSupportBundleCommand = (watchSlug: string, origin: string) => {
  console.log(watchSlug, origin);
  return useQuery("supportBundleCommand", () => {
    fetchSupportBundleCommand(watchSlug, origin);
  });
};

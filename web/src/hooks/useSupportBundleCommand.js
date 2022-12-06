import axios from "axios";

import { Utilities } from "@src/utilities/utilities";
import { useQuery } from "react-query";

const fetchSupportBundleCommand = async (watchSlug, body) => {
  const res = await fetch(
    `${process.env.API_ENDPOINT}/troubleshoot/app/${watchSlug}/supportbundlecommand`,
    {
      method: "POST",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        origin: window.location.origin,
      }),
    }
  );
  if (!res.ok) {
    throw new Error(`Unexpected status code: ${res.status}`);
  }
  return res.json();
};

export const useSupportBundleCommand = (watchSlug, body) => {
  return useQuery("supportBundleCommand", () => {
    fetchSupportBundleCommand(watchSlug, body);
  });
};

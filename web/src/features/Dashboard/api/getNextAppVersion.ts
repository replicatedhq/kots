import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";
import axios from "axios";

axios.defaults.withCredentials = true;

export const getNextAppVersion = async (appSlug: string) => {
  const config = {
    headers: {
      Authorization: Utilities.getToken(),
      "Content-Type": "application/json",
    },
  };
  try {
    const res = await axios.get(
      `${process.env.API_ENDPOINT}/app/${appSlug}/next-app-version`,
      config
    );

    if (res.status === 200) {
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

export const useNextAppVersion = () => {
  const selectedApp = useSelectedApp();
  return useQuery(
    ["getNextAppVersion", selectedApp?.slug],
    () => getNextAppVersion(selectedApp?.slug || ""),
    {
      /// might want to disable the fetch on window focus for this one
      refetchInterval: 5000,
    }
  );
};

export default { useNextAppVersion };

// This hook has not been integrated yet.
import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import axios from "axios";

export const getLicense = async (appSlug: string) => {
  const config = {
    headers: {
      Authorization: Utilities.getToken(),
      "Content-Type": "application/json"
    }
  };
  try {
    console.log("trying to get license");
    const res = await axios.get(
      `${process.env.API_ENDPOINT}/app/${appSlug}/license`,
      config
    );

    if (res.status === 200) {
      return res.data;
    } else {
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

export const useLicense = (params: string) => {
  return useQuery(["license", params], () => getLicense(params), {
    /// might want to disable the fetch on window focus for this one
    // how to handle data that previous exists in cache
    refetchInterval: 60000,
    refetchIntervalInBackground: false
  });
};

export default useLicense;

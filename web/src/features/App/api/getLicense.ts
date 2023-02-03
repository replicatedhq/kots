// This hook has not been integrated yet.
import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useSelectedApp } from "@features/App";
import axios from "axios";
import { AppLicense } from "@types";

export const getLicense = async ({
  appSlug,
}: {
  appSlug: string;
}): Promise<{
  license: AppLicense;
  success: boolean;
  error: string;
} | null | void> => {
  const config = {
    headers: {
      Authorization: Utilities.getToken(),
      "Content-Type": "application/json",
    },
  };
  try {
    const res = await axios.get(
      `${process.env.API_ENDPOINT}/app/${appSlug}/license`,
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

export const useLicense = () => {
  const { selectedApp } = useSelectedApp();
  return useQuery(["license", selectedApp?.slug], () =>
    getLicense({ appSlug: selectedApp?.slug || "" })
  );
};

export default useLicense;

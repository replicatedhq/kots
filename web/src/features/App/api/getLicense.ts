// This hook has not been integrated yet.
import { useQuery } from '@tanstack/react-query';
import { useSelectedApp } from "@features/App";
import axios from "axios";
import { AppLicense } from "@types";

axios.defaults.withCredentials = true;

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
      "Content-Type": "application/json",
    },
    withCredentials: true,
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
  const selectedApp = useSelectedApp();
  return useQuery(["license", selectedApp?.slug], () =>
    getLicense({ appSlug: selectedApp?.slug || "" })
  );
};

export default useLicense;

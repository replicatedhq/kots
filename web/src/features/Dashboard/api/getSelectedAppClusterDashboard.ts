import { useQuery } from "@tanstack/react-query";
import { useSelectedApp } from "@features/App";
import { DashboardResponse } from "@types";
import axios from "axios";

axios.defaults.withCredentials = true;

export const getSelectedAppClusterDashboard = async ({
  appSlug,
  clusterId,
}: {
  appSlug: string;
  clusterId: string;
}): Promise<DashboardResponse | void> => {
  const config = {
    headers: {
      "Content-Type": "application/json",
    },
    withCredentials: true,
  };
  try {
    const res = await axios.get(
      `${process.env.API_ENDPOINT}/app/${appSlug}/cluster/${clusterId}/dashboard`,
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

export const useSelectedAppClusterDashboard = ({
  refetchInterval = false,
}: {
  refetchInterval: number | false;
}) => {
  const selectedApp = useSelectedApp();
  const { slug } = selectedApp || { slug: "" };
  const clusterId = selectedApp?.downstream?.cluster?.id.toString() || "";
  return useQuery(
    ["getAppClusterDashboared", slug, clusterId],
    () =>
      getSelectedAppClusterDashboard({
        appSlug: slug,
        clusterId,
      }),
    {
      refetchInterval,
    }
  );
};

export default { useSelectedAppClusterDashboard };

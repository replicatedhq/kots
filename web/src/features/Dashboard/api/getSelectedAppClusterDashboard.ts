import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useIsHelmManaged } from "@components/hooks";
import { useSelectedApp } from "@features/App";
import { DashboardResponse } from "@types";
import axios from "axios";

axios.defaults.withCredentials = true;

export const getSelectedAppClusterDashboard = async ({
  appSlug,
  clusterId,
  isHelmManaged,
}: {
  appSlug: string;
  clusterId: string;
  isHelmManaged: boolean;
}): Promise<DashboardResponse | void> => {
  const config = {
    headers: {
      Authorization: Utilities.getToken(),
      "Content-Type": "application/json",
    },
  };
  const clusterIdToQuery = isHelmManaged && clusterId === "" ? 0 : clusterId;
  try {
    const res = await axios.get(
      `${process.env.API_ENDPOINT}/app/${appSlug}/cluster/${clusterIdToQuery}/dashboard`,
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
  const { data: isHelmManaged = false } = useIsHelmManaged();

  const selectedApp = useSelectedApp();
  const { slug } = selectedApp || { slug: "" };
  const clusterId = selectedApp?.downstream?.cluster?.id.toString() || "";
  return useQuery(
    ["getAppClusterDashboared", slug, clusterId],
    () =>
      getSelectedAppClusterDashboard({
        appSlug: slug,
        clusterId,
        isHelmManaged,
      }),
    {
      refetchInterval,
    }
  );
};

export default { useSelectedAppClusterDashboard };

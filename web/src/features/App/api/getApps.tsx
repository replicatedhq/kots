// This hook has not been integrated yet.
import React from "react";
import { useQuery, UseQueryResult } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { App } from "@types";

async function getApps({
  accessToken = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
} = {}): Promise<{ apps: App[] } | null> {
  try {
    const res = await _fetch(`${apiEndpoint}/apps`, {
      headers: {
        Authorization: accessToken || "",
        "Content-Type": "application/json",
      },
      method: "GET",
      credentials: "include",
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return null;
      }
      throw Error(`Failed to fetch apps with status ${res.status}`);
    }
    return await res.json();
  } catch (err) {
    if (err instanceof Error) {
      throw err;
    }

    throw Error(`Failed to fetch apps with error ${err}`);
  }
}

function useApps({
  _getApps = getApps,
  refetchInterval = false,
}: {
  _getApps?: typeof getApps;
  refetchInterval?: number | false;
} = {}): UseQueryResult<{
  apps: App[] | null;
}> {
  const query: UseQueryResult<{
    apps: App[] | null;
  }> = useQuery("apps", () => _getApps(), {
    refetchInterval,
  });

  return query;
}

function UseApps({
  children,
}: {
  children: (
    props: UseQueryResult<{ apps: App[] | null }, Error>
  ) => React.ReactNode;
}) {
  const query = useApps();

  // TODO: figure this out
  // @ts-ignore
  return children(query);
}

export { getApps, useApps, UseApps };

import { useQuery } from "@tanstack/react-query";

type UpgradeInfoResponse = {
  isConfigurable: boolean;
  hasPreflight: boolean;
  isEC2Install: boolean;
};

type UpgradeInfoParams = {
  api?: string;
  retry?: number;
  slug: string;
};

// Set the retries to 0 when testing
const DEFAULT_RETRY = process.env.NODE_ENV === "test" ? 0 : 3;

async function getUpgradeInfo({
  api = process.env.API_ENDPOINT,
  slug,
}: UpgradeInfoParams): Promise<UpgradeInfoResponse> {
  const jsonResponse = await fetch(`${api}/upgrade-service/app/${slug}`, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
  });

  if (!jsonResponse.ok) {
    throw new Error(
      `Encountered an error while upgrade info: Unexpected status code: ${jsonResponse.status}`
    );
  }

  try {
    const response: UpgradeInfoResponse = await jsonResponse.json();

    return response;
  } catch (err) {
    console.error(err);
    throw new Error("Encountered an error while unmarshalling upgrade info");
  }
}

function useGetUpgradeInfo({
  slug,
  api,
  retry = DEFAULT_RETRY,
}: UpgradeInfoParams) {
  return useQuery({
    queryFn: () => getUpgradeInfo({ slug, api }),
    queryKey: ["upgrade-info", slug],
    retry,
    onError: (err: Error) => {
      console.log(err);
    },
  });
}

export { useGetUpgradeInfo };

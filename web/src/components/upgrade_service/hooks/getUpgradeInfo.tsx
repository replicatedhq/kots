import { useQuery } from "@tanstack/react-query";

type UpgradeInfoResponse = {
  isConfigurable: boolean;
  hasPreflight: boolean;
};

async function getUpgradeInfo({
  slug,
}: {
  slug: string;
}): Promise<UpgradeInfoResponse> {
  const jsonResponse = await fetch(
    `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}`,
    {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
    }
  );

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

function useGetUpgradeInfo({ slug }: { slug: string }) {
  return useQuery({
    queryFn: () => getUpgradeInfo({ slug }),
    queryKey: ["upgrade-info", slug],
    onError: (err: Error) => {
      console.log(err);
    },
  });
}

export { useGetUpgradeInfo };

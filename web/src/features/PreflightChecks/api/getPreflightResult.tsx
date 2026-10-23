import { usePreflightResults } from "./usePreflightResults";

export function useGetPrelightResults({ slug, sequence }: {
  slug: string;
  sequence?: string;
}) {
  const appUrl = `${process.env.API_ENDPOINT}/app/${slug}`;
  return usePreflightResults({
    url: sequence ? `${appUrl}/sequence/${sequence}/preflight/result` : `${appUrl}/preflight/result`,
    queryKey: ["preflight-results", sequence, slug],
    staleTime: 500,
  });
}

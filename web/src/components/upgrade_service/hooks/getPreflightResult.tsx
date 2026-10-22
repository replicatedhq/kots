import { usePreflightResults } from "@features/PreflightChecks/api/usePreflightResults";

export function useGetPrelightResults({ slug, sequence, enabled = true }: {
  slug: string;
  sequence?: string;
  enabled?: boolean;
}) {
  return usePreflightResults({
    url: `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/preflight/result`,
    queryKey: ["preflight-results", sequence, slug],
    enabled,
  });
}

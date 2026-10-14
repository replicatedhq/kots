import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { ReactNode } from "react";
import { useGetPrelightResults as useInstallResults } from "./getPreflightResult";
import { useGetPrelightResults as useUpgradeResults } from "@components/upgrade_service/hooks/getPreflightResult";

describe.each([
  ["installation", useInstallResults],
  ["upgrade", useUpgradeResults],
])("%s preflight error display", (_name, useResults) => {
  const originalFetch = global.fetch;
  afterEach(() => { global.fetch = originalFetch; });

  it.each([
    ["collection error", false, true, true],
    ["permissions error", true, true, true],
    ["existing error without notices", false, false, true],
    ["successful run with a notice", false, true, false],
  ])("%s", async (_case, isRbac, withNotice, withError) => {
    let recovered = false;
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        preflightResult: {
          skipped: false,
          hasFailingStrictPreflights: withError && !recovered,
          result: {
            errors: withError && !recovered ? [{ error: "Collection failed", isRbac }] : [],
            notice: withNotice ? "Unsupported v1beta3 specs" : undefined,
            results: recovered ? [{ isPass: true, title: "Manual check passed" }] : [],
          },
        },
      }),
    });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    try {
      const { result, unmount } = renderHook(() => useResults({ slug: "test-app" }), { wrapper });
      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data?.showPreflightResultErrors).toBe(withError);
      expect(result.current.data?.showPreflightResults).toBe(!withError && withNotice);
      expect(result.current.data?.showRbacError).toBe(withError && isRbac);
      expect(result.current.data?.errors).toEqual(withError ? ["Collection failed"] : []);
      expect(result.current.data?.pollForUpdates).toBe(withError);
      expect(result.current.data?.showDeploymentBlocked).toBe(withError);
      if (withError) {
        recovered = true;
        await waitFor(() => expect(result.current.data?.errors).toEqual([]), { timeout: 3000 });
        expect(result.current.data?.preflightResults.some(check => check.showPass)).toBe(true);
        expect(result.current.data?.pollForUpdates).toBe(false);
        expect(result.current.data?.showDeploymentBlocked).toBe(false);
      }
      unmount();
    } finally {
      client.clear();
    }
  });
});

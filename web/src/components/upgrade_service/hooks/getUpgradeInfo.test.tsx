/**
 * @jest-environment jest-fixed-jsdom
 */
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react-hooks";
import { useGetUpgradeInfo } from "./getUpgradeInfo";
import { ReactElement } from "react";
import { getSlug } from "@src/utilities/test-utils";

describe("useGetUpgradeInfo", () => {
  const api = "http://test-api";
  const server = setupServer();
  let queryClient: QueryClient;
  let wrapper: ({ children }: { children: ReactElement }) => ReactElement;

  beforeAll(() => {
    server.listen();
  });

  afterAll(() => {
    server.close();
  });

  afterEach(() => {
    // Remove any handlers added
    // in individual tests (runtime handlers).
    server.resetHandlers();
  });

  beforeEach(() => {
    queryClient = new QueryClient();
    wrapper = function wrapperFunc({ children }) {
      return (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      );
    };
  });

  it("normal response", async () => {
    const slug = getSlug(expect);
    server.use(
      http.get(`${api}/upgrade-service/app/${slug}`, () => {
        return HttpResponse.json({
          isConfigurable: true,
          hasPreflight: false,
        });
      })
    );

    const { result, waitFor } = renderHook(useGetUpgradeInfo, {
      initialProps: { api, slug },
      // @ts-expect-error: struggling to make the wrapper types comply, ignoring for now
      wrapper,
    });

    await waitFor(() => result.current.isSuccess);
    expect(result.current.data.isConfigurable).toStrictEqual(true);
    expect(result.current.data.hasPreflight).toStrictEqual(false);
  });

  it("non JSON response throws an error and is handled by the hook", async () => {
    const slug = getSlug(expect);
    server.use(
      http.get(`${api}/upgrade-service/app/${slug}`, () => {
        return HttpResponse.text("this should produce an error");
      })
    );

    const { result, waitFor } = renderHook(useGetUpgradeInfo, {
      initialProps: { api, slug, retry: 0 },
      // @ts-expect-error: struggling to make the wrapper types comply, ignoring for now
      wrapper,
    });

    await waitFor(() => result.current.isError);
    expect(result.current.error).toBeDefined();
  });

  it("4xx response throws an error and is handled by the hook", async () => {
    const slug = getSlug(expect);
    server.use(
      http.get(`${api}/upgrade-service/app/${slug}`, () => {
        return new HttpResponse("Not found", { status: 404 });
      })
    );

    const { result, waitFor } = renderHook(useGetUpgradeInfo, {
      initialProps: { api, slug, retry: 0 },
      // @ts-expect-error: struggling to make the wrapper types comply, ignoring for now
      wrapper,
    });

    await waitFor(() => result.current.isError);
    expect(result.current.error).toBeDefined();
  });

  it("5xx response throws an error and is handled by the hook", async () => {
    const slug = getSlug(expect);
    server.use(
      http.get(`${api}/upgrade-service/app/${slug}`, () => {
        return new HttpResponse("Something is really broken", { status: 503 });
      })
    );

    const { result, waitFor } = renderHook(useGetUpgradeInfo, {
      initialProps: { api, slug, retry: 0 },
      // @ts-expect-error: struggling to make the wrapper types comply, ignoring for now
      wrapper,
    });

    await waitFor(() => result.current.isError);
    expect(result.current.error).toBeDefined();
  });
});

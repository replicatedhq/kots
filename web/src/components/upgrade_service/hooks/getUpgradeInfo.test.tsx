/**
 * @jest-environment jest-fixed-jsdom
 */
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor, act } from "@testing-library/react";
import { useGetUpgradeInfo } from "./getUpgradeInfo";
import { ReactElement } from "react";
import { getSlug } from "@src/utilities/test-utils";

describe("useGetUpgradeInfo", () => {
  const api = "http://test-api";
  const server = setupServer();
  let queryClient: QueryClient;
  let wrapper: ({ children }: { children: ReactElement }) => ReactElement;

  beforeAll(() => {
    server.listen({
      onUnhandledRequest: 'error'
    });
  });

  afterAll(() => {
    server.close();
  });

  afterEach(() => {
    // Remove any handlers added
    // in individual tests (runtime handlers).
    server.resetHandlers();
    queryClient.clear();
  });

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
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
    const expectedUrl = `${api}/upgrade-service/app/${slug}`;

    server.resetHandlers(
      http.get(expectedUrl, () => {
        return HttpResponse.json({
          isConfigurable: true,
          hasPreflight: false,
        });
      })
    );

    const { result } = renderHook(useGetUpgradeInfo, {
      initialProps: { slug, api },
      wrapper,
    });

    // Allow the query to initialize and complete
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 100));
    });

    // Wait for the query to complete
    await waitFor(() => result.current.isSuccess || result.current.isError, { timeout: 5000 });

    expect(result.current.isSuccess).toBe(true);
    expect(result.current.data).toBeDefined();
    expect(result.current.data?.isConfigurable).toStrictEqual(true);
    expect(result.current.data?.hasPreflight).toStrictEqual(false);
  });

  it("non JSON response throws an error and is handled by the hook", async () => {
    const slug = getSlug(expect);

    server.use(
      http.get(`${api}/upgrade-service/app/${slug}`, () => {
        return HttpResponse.text("this should produce an error");
      })
    );

    const { result } = renderHook(useGetUpgradeInfo, {
      initialProps: { slug, retry: 0 },
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

    const { result } = renderHook(useGetUpgradeInfo, {
      initialProps: { slug, retry: 0 },
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

    const { result } = renderHook(useGetUpgradeInfo, {
      initialProps: { slug, retry: 0 },
      wrapper,
    });

    await waitFor(() => result.current.isError);
    expect(result.current.error).toBeDefined();
  });

});

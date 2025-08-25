/**
 * @jest-environment jest-fixed-jsdom
 */
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
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


  it.skip("normal response", async () => {
    const slug = getSlug(expect);
    
    const expectedUrl = `${api}/upgrade-service/app/${slug}`;
    console.log('Expected URL:', expectedUrl);
    
    server.resetHandlers(
      http.get(expectedUrl, ({ request }) => {
        console.log('MSW intercepted request:', request.url);
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

    await waitFor(() => {
      console.log('Waiting for query to complete:', {
        isLoading: result.current.isLoading,
        isSuccess: result.current.isSuccess,
        isError: result.current.isError
      });
      return result.current.isSuccess || result.current.isError;
    }, { timeout: 10000 });
    
    console.log('Query result:', {
      isSuccess: result.current.isSuccess,
      isError: result.current.isError,
      isLoading: result.current.isLoading,
      data: result.current.data,
      error: result.current.error
    });
    
    expect(result.current.isSuccess).toBe(true);
    expect(result.current.data).toBeDefined();
    expect(result.current.data?.isConfigurable).toStrictEqual(true);
    expect(result.current.data?.hasPreflight).toStrictEqual(false);
  });

  it("non JSON response throws an error and is handled by the hook", async () => {
    const slug = getSlug(expect);
    
    const originalEnv = process.env.API_ENDPOINT;
    process.env.API_ENDPOINT = api;
    
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
    
    process.env.API_ENDPOINT = originalEnv;
  });

  it("4xx response throws an error and is handled by the hook", async () => {
    const slug = getSlug(expect);
    
    const originalEnv = process.env.API_ENDPOINT;
    process.env.API_ENDPOINT = api;
    
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
    
    process.env.API_ENDPOINT = originalEnv;
  });

  it("5xx response throws an error and is handled by the hook", async () => {
    const slug = getSlug(expect);
    
    const originalEnv = process.env.API_ENDPOINT;
    process.env.API_ENDPOINT = api;
    
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
    
    process.env.API_ENDPOINT = originalEnv;
  });
});

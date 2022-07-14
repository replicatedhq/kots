/**
 * @jest-environment jsdom
 */
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";
import { renderHook } from "@testing-library/react-hooks";
import { useIsHelmManaged, fetchIsHelmManaged } from "./useIsHelmManaged";

describe("useIsHelmManaged", () => {
  describe("useIsHelmManaged", () => {
    let queryClient;
    let wrapper;
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
    it("calls _fetchIsHelmManaged", async () => {
      const fetchIsHelmManagedSpy = jest.fn(() => Promise.resolve());

      const { result, waitFor } = renderHook(
        () => useIsHelmManaged({ _fetchIsHelmManaged: fetchIsHelmManagedSpy }),
        {
          wrapper,
        }
      );

      await waitFor(() => result.current.isSuccess);

      expect(result.current.variables).toEqual(undefined);
      expect(fetchIsHelmManagedSpy).toHaveBeenCalledTimes(1);
      expect(fetchIsHelmManagedSpy).toHaveBeenCalledWith();
    });
  });
  describe("fetchIsHelmManaged", () => {
    it("calls fetch with the correct url and configuration", async () => {
      const expectedBody = {
        isHelmManaged: true,
      };
      const jsonSpy = jest.fn(() => Promise.resolve(expectedBody));
      const fetchIsHelmManagedSpy = jest.fn(() =>
        Promise.resolve({
          ok: true,
          json: jsonSpy,
        })
      );
      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testFetchIsHelmManagedConfig = {
        _fetch: fetchIsHelmManagedSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      const expectedAPIEndpoint = `${testAPIEndpoint}/isHelmManaged`;
      const expectedResponse = {
        isHelmManaged: true,
      };
      const expectedFetchConfig = {
        method: "GET",
        headers: {
          Authorization: testToken,
          "Content-Type": "application/json",
        },
      };
      await expect(
        fetchIsHelmManaged(testFetchIsHelmManagedConfig)
      ).resolves.toEqual(expectedResponse);
      expect(fetchIsHelmManagedSpy).toHaveBeenCalledTimes(1);
      expect(fetchIsHelmManagedSpy).toHaveBeenCalledWith(
        expectedAPIEndpoint,
        expectedFetchConfig
      );
      expect(jsonSpy).toHaveBeenCalledTimes(1);
    });

    it("throws error when response is not ok", async () => {
      const fetchIsHelmManagedSpy = jest.fn(() =>
        Promise.resolve({
          ok: false,
        })
      );
      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testFetchIsHelmManagedConfig = {
        _fetch: fetchIsHelmManagedSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(
        fetchIsHelmManaged(testFetchIsHelmManagedConfig)
      ).rejects.toThrowError("Error fetching isHelmManaged");
    });

    it("throws error when response is not json", async () => {
      const fetchIsHelmManagedSpy = jest.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.reject(new Error("Error parsing json")),
        })
      );

      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testFetchIsHelmManagedConfig = {
        _fetch: fetchIsHelmManagedSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(
        fetchIsHelmManaged(testFetchIsHelmManagedConfig)
      ).rejects.toThrowError("Error parsing json");
    });
    it("throws error when network error", async () => {
      const fetchIsHelmManagedSpy = jest.fn(() =>
        Promise.reject(new Error("Error fetching"))
      );

      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testFetchIsHelmManagedConfig = {
        _fetch: fetchIsHelmManagedSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(
        fetchIsHelmManaged(testFetchIsHelmManagedConfig)
      ).rejects.toThrowError("Error fetching");
    });
  });
});

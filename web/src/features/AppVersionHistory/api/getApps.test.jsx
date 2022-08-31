/**
 * @jest-environment jsdom
 */
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";
import { renderHook } from "@testing-library/react-hooks";
import { getApps, useApps } from "./getApps";

describe("getApps", () => {
  describe("useApps", () => {
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
    it("calls _getApps", async () => {
      const getAppsSpy = jest.fn(() => Promise.resolve());

      const { result, waitFor } = renderHook(
        () => useApps({ _getApps: getAppsSpy }),
        {
          wrapper,
        }
      );

      await waitFor(() => result.current.isSuccess);

      expect(result.current.variables).toEqual(undefined);
      expect(getAppsSpy).toHaveBeenCalledTimes(1);
      expect(getAppsSpy).toHaveBeenCalledWith();
    });
  });
  describe("getAppsFetch", () => {
    it("calls fetch with the correct url and configuration", async () => {
      const expectedBody = {
        apps: "myapps",
      };
      const jsonSpy = jest.fn(() => Promise.resolve(expectedBody));
      const getAppsSpy = jest.fn(() =>
        Promise.resolve({
          ok: true,
          json: jsonSpy,
        })
      );
      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testgetAppsFetchConfig = {
        _fetch: getAppsSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      const expectedAPIEndpoint = `${testAPIEndpoint}/apps`;
      const expectedFetchConfig = {
        method: "GET",
        headers: {
          Authorization: testToken,
          "Content-Type": "application/json",
        },
      };
      await expect(getApps(testgetAppsFetchConfig)).resolves.toEqual(
        expectedBody
      );
      expect(getAppsSpy).toHaveBeenCalledTimes(1);
      expect(getAppsSpy).toHaveBeenCalledWith(
        expectedAPIEndpoint,
        expectedFetchConfig
      );
      expect(jsonSpy).toHaveBeenCalledTimes(1);
    });

    it("throws error when response is not ok", async () => {
      const getAppsSpy = jest.fn(() =>
        Promise.resolve({
          ok: false,
          status: 400,
        })
      );
      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testgetAppsFetchConfig = {
        _fetch: getAppsSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(getApps(testgetAppsFetchConfig)).rejects.toThrowError(
        "Failed to fetch apps with status 400"
      );
    });

    it("throws error when response is not json", async () => {
      const getAppsSpy = jest.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.reject(new Error("Error parsing json")),
        })
      );

      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testgetAppsFetchConfig = {
        _fetch: getAppsSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(getApps(testgetAppsFetchConfig)).rejects.toThrowError(
        "Error parsing json"
      );
    });
    it("throws error when network error", async () => {
      const getAppsSpy = jest.fn(() =>
        Promise.reject(new Error("Error fetching"))
      );

      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testgetAppsFetchConfig = {
        _fetch: getAppsSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(getApps(testgetAppsFetchConfig)).rejects.toThrowError(
        "Error fetching"
      );
    });
  });
});

/**
 * @jest-environment jsdom
 */
import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react-hooks";
import { useSaveConfig, putConfig } from "./useSaveConfig";

describe("useSaveConfig", () => {
  describe("useSaveConfig", () => {
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
    it("calls _putConfig", async () => {
      const putConfigSpy = jest.fn(() => Promise.resolve());

      const testBody = {
        test: "test",
      };
      const testConfig = {
        appSlug: "test",
        _putConfig: putConfigSpy,
      };
      const { result, waitFor } = renderHook(() => useSaveConfig(testConfig), {
        wrapper,
      });

      result.current.mutate({ body: testBody });

      await waitFor(() => result.current.isSuccess);

      expect(result.current.variables).toEqual({ body: testBody });
      expect(putConfigSpy).toHaveBeenCalledTimes(1);
      expect(putConfigSpy).toHaveBeenCalledWith({
        appSlug: testConfig.appSlug,
        body: testBody,
      });
    });
  });
  describe("putConfig", () => {
    it("calls putConfig with the correct url and configuration", async () => {
      const testBody = JSON.stringify({
        test: "test",
      });
      const jsonSpy = jest.fn(() => Promise.resolve(testBody));
      const testFetch = jest.fn(() =>
        Promise.resolve({
          ok: true,
          json: jsonSpy,
        })
      );
      const testAppSlug = "testAppSlug";
      const testAPIEndpoint = "testAPIEndpoint";
      const testPutConfig = {
        _fetch: testFetch,
        appSlug: testAppSlug,
        apiEndpoint: testAPIEndpoint,
        body: testBody,
      };

      const expectedAPIEndpoint = `${testAPIEndpoint}/app/${testAppSlug}/config`;
      const expectedResponse = {
        data: testBody,
      };
      const expectedFetchConfig = {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: testBody,
      };
      await expect(putConfig(testPutConfig)).resolves.toEqual(expectedResponse);
      expect(testFetch).toHaveBeenCalledTimes(1);
      expect(testFetch).toHaveBeenCalledWith(
        expectedAPIEndpoint,
        expectedFetchConfig
      );
      expect(jsonSpy).toHaveBeenCalledTimes(1);
    });

    it("throws error when response is not ok", async () => {
      const testBody = JSON.stringify({
        test: "test",
      });
      const testFetch = jest.fn(() =>
        Promise.resolve({
          ok: false,
        })
      );
      const testAppSlug = "testAppSlug";
      const testAPIEndpoint = "testAPIEndpoint";
      const testPutConfig = {
        _fetch: testFetch,
        appSlug: testAppSlug,
        apiEndpoint: testAPIEndpoint,
        body: testBody,
      };

      await expect(putConfig(testPutConfig)).rejects.toThrowError(
        "Error saving config"
      );
    });

    it("throws error when response is not json", async () => {
      const testBody = JSON.stringify({
        test: "test",
      });

      const testFetch = jest.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.reject(new Error("Error parsing json")),
        })
      );

      const testAppSlug = "testAppSlug";
      const testAPIEndpoint = "testAPIEndpoint";
      const testPutConfig = {
        _fetch: testFetch,
        appSlug: testAppSlug,
        apiEndpoint: testAPIEndpoint,
        body: testBody,
      };

      await expect(putConfig(testPutConfig)).rejects.toThrowError(
        "Error parsing json"
      );
    });
    it("throws error when network error", async () => {
      const testFetch = jest.fn(() =>
        Promise.reject(new Error("Error fetching"))
      );

      const testAppSlug = "testAppSlug";
      const testAPIEndpoint = "testAPIEndpoint";
      const testPutConfig = {
        _fetch: testFetch,
        appSlug: testAppSlug,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(putConfig(testPutConfig)).rejects.toThrowError(
        "Error fetching"
      );
    });
  });
});

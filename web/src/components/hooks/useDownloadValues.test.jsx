/**
 * @jest-environment jsdom
 */
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";
import { act, renderHook } from "@testing-library/react-hooks";
import { getValues, useDownloadValues } from "./useDownloadValues";

describe("useDownloadValues", () => {
  describe("GET", () => {
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
    it("calls _getValues", async () => {
      const fetchValuesSpy = jest.fn(() => Promise.resolve());
      const testConfig = {
        _getValues: fetchValuesSpy,
        _createObjectURL: jest.fn(() => "test"),
        _revokeObjectURL: jest.fn(() => "test"),
        appSlug: "test",
        fileName: "test",
        sequence: 1,
        vaersionLabel: "1.2.3",
        isPending: false,
      };

      const { result, waitFor } = renderHook(
        () => useDownloadValues(testConfig),
        {
          wrapper,
        }
      );

      await act(async () => {
        await result.current.download();
      });
      await waitFor(() => result.current.isSuccess);

      const expectedFetchConfig = {
        appSlug: testConfig.appSlug,
        sequence: testConfig.sequence,
        versionLabel: testConfig.versionLabel,
        isPending: testConfig.isPending,
      };

      expect(result.current.variables).toEqual(undefined);
      expect(fetchValuesSpy).toHaveBeenCalledTimes(1);
      expect(fetchValuesSpy).toHaveBeenCalledWith(expectedFetchConfig);
    });
    it.todo(
      "test the rest of this or figure out how to use react-query with it"
    );
  });
  describe("getValues", () => {
    it("calls getValues with the correct url and configuration", async () => {
      const expectedBody = {
        test: "test",
      };
      const blobSpy = jest.fn(() => Promise.resolve(expectedBody));
      const _fetchValuesSpy = jest.fn(() =>
        Promise.resolve({
          ok: true,
          blob: blobSpy,
        })
      );
      const testAppSlug = "testAppSlug";
      const testSequence = 1;
      const versionLabel = "1.2.3";
      const isPending = false;
      const testAPIEndpoint = "testAPIEndpoint";
      const testGetValuesConfig = {
        _fetch: _fetchValuesSpy,
        apiEndpoint: testAPIEndpoint,
        appSlug: testAppSlug,
        sequence: testSequence,
        versionLabel: versionLabel,
        isPending: isPending,
      };

      const expectedAPIEndpoint = `${testAPIEndpoint}/app/${testAppSlug}/values/${testSequence}?isPending=${isPending}&semver=${versionLabel}`;
      const expectedResponse = {
        data: expectedBody,
      };
      const expectedFetchConfig = {
        method: "GET",
        headers: {
          "Content-Type": "application/blob",
        },
        credentials: "include",
      };
      await expect(getValues(testGetValuesConfig)).resolves.toEqual(
        expectedResponse
      );
      expect(_fetchValuesSpy).toHaveBeenCalledTimes(1);
      expect(_fetchValuesSpy).toHaveBeenCalledWith(
        expectedAPIEndpoint,
        expectedFetchConfig
      );
      expect(blobSpy).toHaveBeenCalledTimes(1);
    });

    it("throws error when response is not ok", async () => {
      const _fetchValuesSpy = jest.fn(() =>
        Promise.resolve({
          ok: false,
        })
      );
      const testAppSlug = "testAppSlug";
      const testSequence = 1;
      const testVersionLabel = "1.2.3";
      const testIsPending = false;

      const testAPIEndpoint = "testAPIEndpoint";
      const testGetValuesConfig = {
        _fetch: _fetchValuesSpy,
        _token: testToken,
        apiEndpoint: testAPIEndpoint,
        appSlug: testAppSlug,
        sequence: testSequence,
        versionLabel: testVersionLabel,
        isPending: testIsPending,
      };

      await expect(getValues(testGetValuesConfig)).rejects.toThrowError(
        "Error fetching values"
      );
    });

    it("throws error when response is not blob", async () => {
      const _fetchValuesSpy = jest.fn(() =>
        Promise.resolve({
          ok: true,
          blob: () => Promise.reject(new Error("Error parsing blob")),
        })
      );

      const testAppSlug = "testAppSlug";
      const testSequence = 1;
      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testGetValuesConfig = {
        _fetch: _fetchValuesSpy,
        _token: testToken,
        apiEndpoint: testAPIEndpoint,
        appSlug: testAppSlug,
        sequence: testSequence,
      };

      await expect(getValues(testGetValuesConfig)).rejects.toThrowError(
        "Error parsing blob"
      );
    });
    it("throws error when network error", async () => {
      const _fetchValuesSpy = jest.fn(() =>
        Promise.reject(new Error("Error fetching"))
      );

      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testGetValuesConfig = {
        _fetch: _fetchValuesSpy,
        _token: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(getValues(testGetValuesConfig)).rejects.toThrowError(
        "Error fetching"
      );
    });
  });
});

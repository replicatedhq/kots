/**
 * @jest-environment jsdom
 */
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";
import { renderHook } from "@testing-library/react-hooks";
import {
  getVersions,
  useVersions,
  getVersionsSelectorForHelmManaged,
} from "./getVersions";

describe("getVersions", () => {
  describe("useVersions", () => {
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
    it("calls _getVersions", async () => {
      const getVersionsSpy = jest.fn(() =>
        Promise.resolve({
          versionHistory: [
            {
              status: "deployed",
              sequence: "0",
            },
          ],
        })
      );

      const { result, waitFor } = renderHook(
        () =>
          useVersions({
            _getVersions: getVersionsSpy,
            _useParams: () => ({ slug: "testSlug" }),
            _useCurrentApp: () => ({ currentApp: "testCurrentApp" }),
            _useMetadata: () => ({ data: { isAirGap: false, isKurl: false } }),
            _useIsHelmManaged: () => ({ data: { isHelmManaged: true } }),
          }),
        {
          wrapper,
        }
      );

      await waitFor(() => result.current.isSuccess);

      expect(result.current.variables).toEqual(undefined);
      expect(getVersionsSpy).toHaveBeenCalledTimes(1);
      expect(getVersionsSpy).toHaveBeenCalledWith({ slug: "testSlug" });
    });
  });
  describe("getVersionsFetch", () => {
    it("calls getVersionsFetch with the correct url and configuration", async () => {
      const expectedBody = {
        apps: "myapps",
      };
      const jsonSpy = jest.fn(() => Promise.resolve(expectedBody));
      const getVersionsSpy = jest.fn(() =>
        Promise.resolve({
          ok: true,
          json: jsonSpy,
        })
      );
      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testgetVersionsFetchConfig = {
        _fetch: getVersionsSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
        slug: "testSlug",
        curentPage: 0,
        pageSize: 20,
      };

      const expectedAPIEndpoint = `${testAPIEndpoint}/app/testSlug/versions?currentPage=0&pageSize=20&pinLatestDeployable=true`;
      const expectedFetchConfig = {
        method: "GET",
        headers: {
          Authorization: testToken,
          "Content-Type": "application/json",
        },
      };
      await expect(getVersions(testgetVersionsFetchConfig)).resolves.toEqual(
        expectedBody
      );
      expect(getVersionsSpy).toHaveBeenCalledTimes(1);
      expect(getVersionsSpy).toHaveBeenCalledWith(
        expectedAPIEndpoint,
        expectedFetchConfig
      );
      expect(jsonSpy).toHaveBeenCalledTimes(1);
    });

    it("throws error when response is not ok", async () => {
      const getVersionsSpy = jest.fn(() =>
        Promise.resolve({
          ok: false,
          status: 400,
        })
      );
      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testgetVersionsFetchConfig = {
        _fetch: getVersionsSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(
        getVersions(testgetVersionsFetchConfig)
      ).rejects.toThrowError("Failed to fetch apps with status 400");
    });

    it("throws error when response is not json", async () => {
      const getVersionsSpy = jest.fn(() =>
        Promise.resolve({
          ok: true,
          json: () => Promise.reject(new Error("Error parsing json")),
        })
      );

      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testgetVersionsFetchConfig = {
        _fetch: getVersionsSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(
        getVersions(testgetVersionsFetchConfig)
      ).rejects.toThrowError("Error parsing json");
    });
    it("throws error when network error", async () => {
      const getVersionsSpy = jest.fn(() =>
        Promise.reject(new Error("Error fetching"))
      );

      const testToken = "testToken";
      const testAPIEndpoint = "testAPIEndpoint";
      const testgetVersionsFetchConfig = {
        _fetch: getVersionsSpy,
        accessToken: testToken,
        apiEndpoint: testAPIEndpoint,
      };

      await expect(
        getVersions(testgetVersionsFetchConfig)
      ).rejects.toThrowError("Error fetching");
    });
  });
  describe("getVersionsSelectorForHelmManaged", () => {
    it("returns Redeploy for currently deployed version", () => {
      const data = {
        versionHistory: [
          {
            status: "deployed",
            sequence: "0",
          },
        ],
      };

      const expectedData = {
        versionHistory: [
          {
            status: "deployed",
            sequence: "0",
            statusLabel: "Redeploy",
          },
        ],
      };
      const result = getVersionsSelectorForHelmManaged({ versions: data });
      expect(result).toEqual(expectedData);
    });

    it("returns Deploy and rollback for new and old versions", () => {
      const data = {
        versionHistory: [
          {
            status: "deployed",
            sequence: "1",
          },
          {
            status: "pending",
            sequence: "2",
          },
          {
            status: "pending",
            sequence: "3",
          },
          {
            status: "pending",
            sequence: "0",
          },
        ],
      };

      const expectedData = {
        versionHistory: [
          {
            status: "deployed",
            sequence: "1",
            statusLabel: "Redeploy",
          },
          {
            status: "pending",
            sequence: "2",
            statusLabel: "Deploy",
          },
          {
            status: "pending",
            sequence: "3",
            statusLabel: "Deploy",
          },
          {
            status: "pending",
            sequence: "0",
            statusLabel: "Rollback",
          },
        ],
      };
      const result = getVersionsSelectorForHelmManaged({ versions: data });
      expect(result).toEqual(expectedData);
    });
  });
});

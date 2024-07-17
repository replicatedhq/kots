import { Utilities } from "./utilities";

describe("Utilities", () => {
  describe("checkIsDateExpired", () => {
    it("should return true if date is expired", () => {
      const date = new Date().setMinutes(new Date().getMinutes() - 1);
      const timestamp = new Date(date).toISOString();
      expect(Utilities.checkIsDateExpired(timestamp)).toBe(true);
    });

    it("should return false if date is not expired", () => {
      const date = new Date().setHours(new Date().getHours() + 1);
      const timestamp = new Date(date).toISOString();
      expect(Utilities.checkIsDateExpired(timestamp)).toBe(false);
    });
  });

  describe("isInitialAppInstall", () => {
    it("should return true if app is null", () => {
      expect(Utilities.isInitialAppInstall(null)).toBe(true);
    });

    it("should return false if there is a current version", () => {
      const app = {
        downstream: {
          currentVersion: {
            status: "deployed",
          },
        },
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(false);
    });

    it("should return true if there is no pending versions", () => {
      let app = {
        downstream: {
          pendingVersions: [],
        },
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(true);

      app = {
        downstream: {},
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(true);
    });

    it("should return false if there is more than one pending version", () => {
      const app = {
        downstream: {
          pendingVersions: [
            {
              status: "pending_config",
            },
            {
              status: "pending_config",
            },
          ],
        },
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(false);
    });

    it("should return true if first pending version has status `pending_cluster_management`, `pending_config`, `pending_preflight`, or `pending_download`", () => {
      let app = {
        downstream: {
          pendingVersions: [
            {
              status: "pending_cluster_management",
            },
          ],
        },
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(true);

      app = {
        downstream: {
          pendingVersions: [
            {
              status: "pending_config",
            },
          ],
        },
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(true);

      app = {
        downstream: {
          pendingVersions: [
            {
              status: "pending_preflight",
            },
          ],
        },
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(true);

      app = {
        downstream: {
          pendingVersions: [
            {
              status: "pending_download",
            },
          ],
        },
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(true);
    });

    it("should return false if first pending version does not have status `pending_cluster_management`, `pending_config`, `pending_preflight`, or `pending_download`", () => {
      let app = {
        downstream: {
          pendingVersions: [
            {
              status: "pending",
            },
          ],
        },
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(false);

      app = {
        downstream: {
          pendingVersions: [
            {
              status: "unknown",
            },
          ],
        },
      };
      expect(Utilities.isInitialAppInstall(app)).toBe(false);
    });
  });

  describe("snapshotLocationStr", () => {
    it("should return bucket name if path is empty or undefined", () => {
      expect(Utilities.snapshotLocationStr("my-bucket", "")).toBe("my-bucket");
      expect(Utilities.snapshotLocationStr("my-bucket", undefined)).toBe(
        "my-bucket"
      );
    });

    it("should return bucket name and path if path is not empty", () => {
      expect(Utilities.snapshotLocationStr("my-bucket", "my-path")).toBe(
        "my-bucket/my-path"
      );
    });

    it("should return bucket name and path if path is not empty and begins with a slash", () => {
      expect(Utilities.snapshotLocationStr("my-bucket", "/my-path")).toBe(
        "my-bucket/my-path"
      );
    });

    it("should not error if bucket and path are undefined", () => {
      expect(Utilities.snapshotLocationStr(undefined, undefined)).toBe("");
    });
  });

  describe("getSubnavItemForRoute", () => {
    it("should return an empty string if there is no route", () => {
      expect(Utilities.getSubnavItemForRoute(undefined, "my-app")).toBe("");
    });

    it("should return an empty string if there is no app slug", () => {
      expect(
        Utilities.getSubnavItemForRoute("/app/my-app/config", undefined)
      ).toBe("");
    });

    it("should return an empty string if there is no subnav item", () => {
      expect(Utilities.getSubnavItemForRoute("/app/my-app/", "my-app")).toBe(
        ""
      );
    });

    it("should return the subnav item for the route", () => {
      expect(
        Utilities.getSubnavItemForRoute("/app/my-app/config", "my-app")
      ).toBe("config");
    });

    it("should return the subnav item for the route with a subpath", () => {
      expect(
        Utilities.getSubnavItemForRoute("/app/my-app/config/1", "my-app")
      ).toBe("config");
    });

    it("should return the subnav item for the route with multiple subpaths", () => {
      expect(
        Utilities.getSubnavItemForRoute(
          "/app/my-app/troubleshoot/analyze/abcdefg",
          "my-app"
        )
      ).toBe("troubleshoot");
    });
  });
});

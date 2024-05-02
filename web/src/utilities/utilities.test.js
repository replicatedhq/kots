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

  describe("shouldShowClusterUpgradeModal", () => {
    it("should return false if apps is null or empty", () => {
      expect(Utilities.shouldShowClusterUpgradeModal(null)).toBe(false);
      expect(Utilities.shouldShowClusterUpgradeModal([])).toBe(false);
    });

    it("should return false if the user has not tried to deploy the current version", () => {
      const apps = [
        {
          downstream: {
            currentVersion: {
              status: "pending",
            },
          },
        },
      ];
      expect(Utilities.shouldShowClusterUpgradeModal(apps)).toBe(false);
    });

    it("should return false if the user has tried to deploy the current version, but a cluster upgrade is not required", () => {
      const apps = [
        {
          downstream: {
            currentVersion: {
              status: "deployed",
            },
            cluster: {
              requiresUpgrade: false,
            },
          },
        },
      ];
      expect(Utilities.shouldShowClusterUpgradeModal(apps)).toBe(false);
    });

    it("should return false if the user has tried to deploy the current version and a cluster upgrade is already completed", () => {
      const apps = [
        {
          downstream: {
            currentVersion: {
              status: "deployed",
            },
            cluster: {
              requiresUpgrade: false,
              state: "Installed",
            },
          },
        },
      ];
      expect(Utilities.shouldShowClusterUpgradeModal(apps)).toBe(false);
    });

    it("should return true if the user has tried to deploy the current version and a cluster upgrade is required", () => {
      const apps = [
        {
          downstream: {
            currentVersion: {
              status: "deployed",
            },
            cluster: {
              requiresUpgrade: true,
              state: "Installed",
            },
          },
        },
      ];
      expect(Utilities.shouldShowClusterUpgradeModal(apps)).toBe(true);
    });

    it("should return true if the user has tried to deploy the current version and a cluster upgrade is in progress", () => {
      for (const installationState of [
        "CopyingArtifacts",
        "Enqueued",
        "Installing",
        "AddonsInstalling",
        "PendingChartCreation",
      ]) {
        const apps = [
          {
            downstream: {
              currentVersion: {
                status: "deployed",
              },
              cluster: {
                requiresUpgrade: true,
                state: installationState,
              },
            },
          },
        ];
        expect(Utilities.shouldShowClusterUpgradeModal(apps)).toBe(true);
      }
    });

    it("should return false if there are is one installation that does not have a state", () => {
      const apps = [
        {
          downstream: {
            currentVersion: {
              status: "deployed",
            },
            cluster: {
              requiresUpgrade: false,
              numInstallations: 1,
            },
          },
        },
      ];
      expect(Utilities.shouldShowClusterUpgradeModal(apps)).toBe(false);
    });

    it("should return true if there are multiple installations, but the latest does not have a state", () => {
      const apps = [
        {
          downstream: {
            currentVersion: {
              status: "deployed",
            },
            cluster: {
              requiresUpgrade: false,
              numInstallations: 2,
            },
          },
        },
      ];
      expect(Utilities.shouldShowClusterUpgradeModal(apps)).toBe(true);
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
});

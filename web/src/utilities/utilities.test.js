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
      const apps = [
        {
          downstream: {
            currentVersion: {
              status: "deployed",
            },
            cluster: {
              requiresUpgrade: true,
              state: "Installing",
            },
          },
        },
      ];
      expect(Utilities.shouldShowClusterUpgradeModal(apps)).toBe(true);
    });
  });
});

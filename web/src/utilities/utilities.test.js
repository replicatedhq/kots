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
});

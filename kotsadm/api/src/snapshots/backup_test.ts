import * as _ from "lodash";
import * as fs from "fs";
import * as path from "path";
import { describe, it } from "mocha";
import { expect } from "chai";
import { formatTTL, parseTTL } from "./backup";

describe("ttl", () => {
  const tests = [{
    parsed: { quantity: 1000, unit: "seconds" },
    duration: "1000s",
  }, {
    parsed:  { quantity: 500, unit: "minutes" },
    duration: "500m",
  }, {
    parsed: { quantity: 3, unit: "years" },
    duration: "26298h",
  }, {
    parsed: { quantity: 5, unit: "weeks" },
    duration: "840h",
  }, {
    parsed: { quantity: 2, unit: "weeks" },
    duration: "336h",
  }, {
    parsed: { quantity: 6, unit: "months" },
    duration: "4320h",
  }, {
    parsed: { quantity: 1, unit: "days" },
    duration: "24h",
  }].forEach((test) => {
    it(test.duration, () => {
      expect(formatTTL(test.parsed.quantity, test.parsed.unit)).to.equal(test.duration);
      expect(parseTTL(test.duration)).to.deep.equal(test.parsed);
    });
  });
});

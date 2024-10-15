import slugify from "slugify";

/**
 *  Helper to build slugs based on test names
 */
export function getSlug(expect: jest.Expect) {
  return `slug-${slugify(expect.getState().currentTestName)}`;
}

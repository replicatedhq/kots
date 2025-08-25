import "intersection-observer";
import "@testing-library/jest-dom";

// Set default API endpoint for tests to prevent undefined URLs
process.env.API_ENDPOINT = process.env.API_ENDPOINT || "http://test-api";

// Global cleanup for Jest
afterEach(() => {
  // Clear any timers and intervals
  jest.clearAllTimers();
  jest.clearAllMocks();
});

// Cleanup timers after all tests to prevent worker hanging
afterAll(() => {
  // Clear any remaining timers
  jest.clearAllTimers();
});

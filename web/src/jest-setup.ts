import "intersection-observer";
import "@testing-library/jest-dom";

// Set default API endpoint for tests to prevent undefined URLs
process.env.API_ENDPOINT = process.env.API_ENDPOINT || "http://test-api";

// Mock fetch globally to prevent actual network calls in tests
const mockFetch = jest.fn().mockImplementation(() => 
  Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve({ apps: [] }),
    text: () => Promise.resolve(""),
    headers: new Map(),
    clone: function() { return this; }, // Add clone method for MSW compatibility
  })
);
global.fetch = mockFetch;

// Mock IntersectionObserver
global.IntersectionObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

// Mock ResizeObserver
global.ResizeObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(), // deprecated
    removeListener: jest.fn(), // deprecated
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
});

// Mock window.scrollTo
Object.defineProperty(window, 'scrollTo', {
  writable: true,
  value: jest.fn(),
});

// Mock window.URL.createObjectURL
Object.defineProperty(window.URL, 'createObjectURL', {
  writable: true,
  value: jest.fn(() => 'mocked-url'),
});

// Mock window.URL.revokeObjectURL
Object.defineProperty(window.URL, 'revokeObjectURL', {
  writable: true,
  value: jest.fn(),
});

// Global cleanup for Jest
afterEach(() => {
  // Clear any timers and intervals
  jest.clearAllTimers();
  jest.clearAllMocks();

  // Reset fetch mock
  mockFetch.mockClear();
});

// Cleanup timers after all tests to prevent worker hanging
afterAll(() => {
  // Clear any remaining timers
  jest.clearAllTimers();

  // Reset fetch mock
  mockFetch.mockReset();
});

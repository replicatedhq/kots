module.exports = {
  // Use jsdom environment for DOM testing
  testEnvironment: 'jsdom',
  
  // Setup files to run before tests
  setupFilesAfterEnv: ['<rootDir>/src/jest-setup.ts'],
  
  // Transform files with babel (Jest 30 automatically uses babel-jest)
  transform: {
    '^.+\\.(js|jsx|ts|tsx)$': 'babel-jest',
  },
  
  // Module name mapping for imports
  moduleNameMapper: {
    '\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$': '<rootDir>/__mocks__/fileMock.js',
    '\\.(scss|css|less)$': '<rootDir>/__mocks__/styleMock.js',
    '@components/(.*)$': '<rootDir>/src/components/$1',
    '@features/(.*)$': '<rootDir>/src/features/$1',
    '@stores/(.*)$': '<rootDir>/src/stores/$1',
    '@types': '<rootDir>/src/types/index',
    '@utils': '<rootDir>/src/utilities/index',
    'handlebars': '<rootDir>/handlebars/dist/handlebars.js',
    '@src/(.*)$': '<rootDir>/src/$1'
  },
  
  // Test environment options
  testEnvironmentOptions: {
    customExportConditions: [''],
  },
  
  // Jest worker configuration to prevent process exceptions
  maxWorkers: 1,
  workerIdleMemoryLimit: '512MB',
  
  // Timeout configuration
  testTimeout: 10000,
  
  // Coverage configuration
  collectCoverageFrom: [
    'src/**/*.{js,jsx,ts,tsx}',
    '!src/**/*.d.ts',
    '!src/**/*.stories.{js,jsx,ts,tsx}',
    '!src/**/*.test.{js,jsx,ts,tsx}',
    '!src/**/index.{js,jsx,ts,tsx}',
  ],
  
  // Test file patterns
  testMatch: [
    '<rootDir>/src/**/__tests__/**/*.{js,jsx,ts,tsx}',
    '<rootDir>/src/**/*.{test,spec}.{js,jsx,ts,tsx}',
  ],
  
  // Module file extensions
  moduleFileExtensions: ['js', 'jsx', 'ts', 'tsx', 'json'],
  
  // Clear mocks between tests
  clearMocks: true,
  
  // Restore mocks between tests
  restoreMocks: true,
  
  // Reset modules between tests
  resetModules: true,
  
  // Collect coverage
  collectCoverage: false,
  
  // Coverage directory
  coverageDirectory: 'coverage',
  
  // Coverage reporters
  coverageReporters: ['text', 'lcov', 'html'],
  
  // Coverage thresholds
  coverageThreshold: {
    global: {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80,
    },
  },
  
  // Verbose output
  verbose: true,
  
  // Force exit after tests
  forceExit: true,
  
  // Detect open handles
  detectOpenHandles: true,
};

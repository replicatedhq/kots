/**
 * @jest-environment jsdom
 */
import React from 'react';
import { useSaveConfig } from './useSaveConfig';
import { QueryClient, QueryClientProvider } from 'react-query';
import {
  renderHook,
} from '@testing-library/react-hooks';

describe('useSaveConfig', () => {
  describe('PUT', () => {
    it('calls _putConfig', async () => {
      const putConfig = jest.fn(() => {
        return Promise.resolve();
      });

      const queryClient = new QueryClient();
      const wrapper = ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      );
      const testBody = {
        test: 'test',
      }
      const testConfig = {
        appSlug: 'test',
        _putConfig: putConfig,
      }
      const { result, waitFor } = renderHook(() => useSaveConfig(testConfig), { wrapper });

      result.current.mutate({body: testBody});

      await waitFor(() => result.current.isSuccess);

      expect(result.current.variables).toEqual({ body: testBody });
      expect(putConfig).toHaveBeenCalledTimes(1);
      expect(putConfig).toHaveBeenCalledWith({ appSlug: testConfig.appSlug, body: testBody });
    });
  });
  describe('putConfig', () => {

    // calls fetch
    // returns success
    // throws error when response is not ok
    // throws error when not json
    // thrwos error when network error

  });
});
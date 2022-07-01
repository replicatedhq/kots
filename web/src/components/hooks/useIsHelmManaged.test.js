// /**
//  * @jest-environment jsdom
//  */
// import {
//   act,
//   renderHook,
// } from '@testing-library/react-hooks';
// import {
//   useCustomer,
// } from './useIsHelmManaged';


// describe('useCustomer', () => {
//   describe('GET', () => {
//     it('calls _fetchCustomer and sets the value of email', async () => {
//       const fetchCustomer = jest.fn(() => {
//         return Promise.resolve({ email: 'test@test.com' });
//       });

//       const { result, waitForNextUpdate } = renderHook(() => {
//         return useCustomer({
//           accessToken: 'accessToken',
//           appId: 'appId',
//           customerId: 'customerId',
//           _fetchCustomer: fetchCustomer,
//         });
//       });
//       await act(async () => {
//         await waitForNextUpdate();
//       })

//       expect(result.current.customer.email).toBe('test@test.com');
//       expect(fetchCustomer).toHaveBeenCalledTimes(1);
//     });
//   });
//   describe('PUT', () => {
//     it('calls _fetchCustomer and _updateCustomer when new email is set', async () => {
//       const fetchCustomer = jest.fn(() => {
//         return Promise.resolve({ email: 'nottest@test.com', id: 'customerId' });
//       });
//       const updateCustomerEmail = jest.fn(() => {
//         return Promise.resolve({});
//       });

//       const { result, waitForNextUpdate } = renderHook(() => {
//         return useCustomer({
//           accessToken: 'accessToken',
//           appId: 'appId',
//           customerId: 'customerId',
//           _fetchCustomer: fetchCustomer,
//           _updateCustomerEmail: updateCustomerEmail,
//         });
//       });
//       await act(async () => {
//         await waitForNextUpdate();
//       })

//       await act(async () => {
//         result.current.setNewEmail('test@test.com')
//       })

//       expect(result.current.customer.email).toBe('test@test.com');
//       expect(result.current.newEmailLoading).toBe(false);
//       expect(fetchCustomer).toHaveBeenCalledTimes(2);
//       expect(updateCustomerEmail).toHaveBeenCalledTimes(1);
//     });

//     it('returns an error new email fails to set', async () => {
//       const fetchCustomer = jest.fn(() => {
//         return Promise.resolve({ email: 'test@test.com', id: 'customerId' });
//       });
//       const updateCustomerEmail = jest.fn(() => {
//         throw new Error('404')
//       });

//       const { result, waitForNextUpdate } = renderHook(() => {
//         return useCustomer({
//           accessToken: 'accessToken',
//           appId: 'appId',
//           customerId: 'customerId',
//           _fetchCustomer: fetchCustomer,
//           _updateCustomerEmail: updateCustomerEmail,
//         });
//       });
//       await act(async () => {
//         await waitForNextUpdate();
//       })

//       await act(async () => {
//         result.current.setNewEmail('newemail@test.com')
//         await waitForNextUpdate();
//       })

//       expect(result.current.customer.email).toBe('test@test.com');
//       expect(result.current.newEmailError).toBe(true);
//     });
//   })
// });
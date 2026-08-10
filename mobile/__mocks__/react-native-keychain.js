/**
 * Jest mock for react-native-keychain: an in-memory per-service store so
 * keychain/bridge/shim tests run without the native module.
 */

const store = new Map();

const getGenericPassword = jest.fn(async ({ service }) => {
  return store.get(service) ?? false;
});

const setGenericPassword = jest.fn(async (username, password, { service }) => {
  store.set(service, { username, password, service, storage: 'mock' });
  return { service, storage: 'mock' };
});

const resetGenericPassword = jest.fn(async ({ service }) => {
  store.delete(service);
  return { service, storage: 'mock' };
});

const hasGenericPassword = jest.fn(async ({ service }) => store.has(service));

module.exports = {
  getGenericPassword,
  setGenericPassword,
  resetGenericPassword,
  hasGenericPassword,
  __store: store,
};

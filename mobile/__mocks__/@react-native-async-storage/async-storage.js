/**
 * Jest mock for @react-native-async-storage/async-storage: an in-memory
 * store so bridge/settings tests run without the native module.
 */

const store = new Map();

const AsyncStorage = {
  getItem: jest.fn(async (key) => (store.has(key) ? store.get(key) : null)),
  setItem: jest.fn(async (key, value) => {
    store.set(key, String(value));
  }),
  removeItem: jest.fn(async (key) => {
    store.delete(key);
  }),
  clear: jest.fn(async () => {
    store.clear();
  }),
  getAllKeys: jest.fn(async () => [...store.keys()]),
};

module.exports = AsyncStorage;
module.exports.default = AsyncStorage;

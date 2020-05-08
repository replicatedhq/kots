export type ResolveFunc = (root: {}, params: {}, context: {}, meta: {}) => Promise<{}>;
export type Provider<T> = () => T;
export type Resolver<T> = T | Promise<T> | Provider<T> | Provider<Promise<T>>;

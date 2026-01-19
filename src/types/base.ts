export interface LoadableData<T> {
  data: T;
  loading: boolean;
  error?: string;
}
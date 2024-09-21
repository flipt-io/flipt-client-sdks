import { ClientOptions } from '@flipt-io/flipt-client-browser';

export interface FliptProviderProps {
  namespace?: string;
  options?: ClientOptions;
  children: React.ReactNode;
}

export * from '@flipt-io/flipt-client-browser';

import React, { useState, useEffect } from 'react';
import { FliptContext, useStore } from './useFliptClient';
import type { ClientOptions } from '@flipt-io/flipt-client-js';

export const FliptProvider: React.FC<{
  children: React.ReactNode;
  namespace: string;
  options: ClientOptions;
}> = ({ children, namespace, options }) => {
  const [store] = useState(useStore(namespace, options));
  useEffect(() => {
    store.attach();
    return () => {
      store.detach();
    };
  }, [store]);

  return (
    <FliptContext.Provider value={store}>{children}</FliptContext.Provider>
  );
};

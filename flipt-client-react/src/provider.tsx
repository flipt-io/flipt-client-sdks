import React, { useState, useEffect } from 'react';
import { FliptContext, useStore } from './useFliptClient';
import type { ClientOptions } from '@flipt-io/flipt-client-js';

export const FliptProvider: React.FC<{
  children: React.ReactNode;
  options: ClientOptions;
}> = ({ children, options }) => {
  const [store] = useState(useStore(options));
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

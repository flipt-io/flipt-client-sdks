import React, { useState, useEffect } from 'react';
import { FliptContext, useStore } from './useFliptClient';
import type { FliptProviderOptions } from './useFliptClient';

export const FliptProvider: React.FC<{
  children: React.ReactNode;
  options: FliptProviderOptions;
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

import React, { createContext, useContext } from 'react';
import { useFliptClient } from './useFliptClient';

const FliptContext = createContext<ReturnType<typeof useFliptClient> | null>(
  null
);

export const FliptProvider: React.FC<{
  children: React.ReactNode;
  namespace: string;
  options: { url: string };
}> = ({ children, namespace, options }) => {
  const fliptState = useFliptClient(namespace, options);

  return (
    <FliptContext.Provider value={fliptState}>{children}</FliptContext.Provider>
  );
};

export const useFliptContext = (): ReturnType<typeof useFliptClient> => {
  const context = useContext(FliptContext);
  if (context === null) {
    throw new Error('useFliptContext must be used within a FliptProvider');
  }
  return context;
};

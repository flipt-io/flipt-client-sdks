export * from './core/types';

// Re-export the base class
export { BaseClient } from './core/base';

// Import implementations
import { FliptClient as NodeImpl } from './node';
import { FliptClient as BrowserImpl } from './browser';

// Export implementations with distinct names
export { NodeImpl as NodeFliptClient };
export { BrowserImpl as BrowserFliptClient };

// Create and export the default client based on environment
export const FliptClient =
  typeof window === 'undefined' ? NodeImpl : BrowserImpl;

// Export the type based on the runtime implementation
export type FliptClient = InstanceType<typeof FliptClient>;

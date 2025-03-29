// Re-export all types
export * from './core/types';

// Re-export the base class
export { BaseFliptClient } from './core/base';

// Export the node and browser implementations with distinct names
import { FliptClient as NodeImpl } from './node';
import { FliptClient as BrowserImpl } from './browser';

export { NodeImpl as NodeFliptClient };
export { BrowserImpl as BrowserFliptClient };

// Set default client based on environment
export const FliptClient = typeof window === 'undefined' ? NodeImpl : BrowserImpl; 
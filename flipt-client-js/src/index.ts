export * from './core/types';

// Export the appropriate implementation based on environment
import { FliptClient as BrowserFliptClient } from './browser';
import { FliptClient as NodeFliptClient } from './node';

// Determine which implementation to use
const isNode = typeof window === 'undefined';
export const FliptClient = isNode ? NodeFliptClient : BrowserFliptClient;

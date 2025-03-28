// jest.polyfills.js
/**
 * @note The block below contains polyfills for Node.js globals
 * required for Jest to function when running JSDOM tests.
 * These HAVE to be require's and HAVE to be in this exact
 * order, since "undici" depends on the "TextEncoder" global API.
 */

import { TextDecoder, TextEncoder } from 'node:util';
import { ReadableStream } from 'node:stream/web';
import { clearImmediate } from 'node:timers';

Object.defineProperties(globalThis, {
  TextDecoder: { value: TextDecoder },
  TextEncoder: { value: TextEncoder },
  ReadableStream: { value: ReadableStream },
  clearImmediate: { value: clearImmediate }
});

import { Blob, File } from 'node:buffer';
import { fetch, Headers, FormData, Request, Response } from 'undici';

Object.defineProperties(globalThis, {
  fetch: { value: fetch, writable: true },
  Blob: { value: Blob },
  File: { value: File },
  Headers: { value: Headers },
  FormData: { value: FormData },
  Request: { value: Request },
  Response: { value: Response }
});

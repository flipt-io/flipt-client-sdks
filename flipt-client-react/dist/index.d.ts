import React$1 from 'react';
import { ClientOptions } from '@flipt-io/flipt-client-browser';

interface FliptProviderProps {
    namespace?: string;
    options?: ClientOptions;
    children: React.ReactNode;
}

declare const FliptProvider: React$1.FC<FliptProviderProps>;

export { FliptProvider };

import React from "react";
import { FliptEvaluationClient, EvaluationRequest, BooleanEvaluationResponse, BatchEvaluationResponse, Flag, VariantEvaluationResponse } from "@flipt-io/flipt-client-browser";
import { FliptProviderProps } from "./types";
interface FliptContextType {
    client: FliptEvaluationClient | null;
    isLoading: boolean;
    error: Error | null;
}
export declare const FliptProvider: React.FC<FliptProviderProps>;
export declare const useFliptClient: () => FliptContextType;
export declare const useRefresh: () => () => void;
export declare const useVariantFlag: (flagKey: string, entityId: string, context: {}) => {
    value: VariantEvaluationResponse | null;
    isLoading: boolean;
    error: Error | null;
};
export declare const useBooleanFlag: (flagKey: string, entityId: string, context: {}) => {
    value: BooleanEvaluationResponse | null;
    isLoading: boolean;
    error: Error | null;
};
export declare const useBatchEvaluation: (requests: EvaluationRequest[]) => {
    results: BatchEvaluationResponse | null;
    isLoading: boolean;
    error: Error | null;
};
export declare const useFlags: () => {
    flags: Flag[] | null;
    isLoading: boolean;
    error: Error | null;
};
export * from "./types";

import { BooleanResult, EngineOpts, VariantResult } from './models.js';
export declare class FliptEvaluationClient {
    private engine;
    private fetcher;
    private constructor();
    static init(namespace?: string, engine_opts?: EngineOpts): Promise<FliptEvaluationClient>;
    refresh(): Promise<void>;
    evaluateVariant(flag_key: string, entity_id: string, context: {}): VariantResult;
    evaluateBoolean(flag_key: string, entity_id: string, context: {}): BooleanResult;
}

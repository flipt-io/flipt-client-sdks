"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.FliptEvaluationClient = void 0;
const pkg_1 = require("../pkg");
class FliptEvaluationClient {
    engine;
    fetcher;
    constructor(engine, fetcher) {
        this.engine = engine;
        this.fetcher = fetcher;
    }
    static async init(namespace = 'default', engine_opts = {
        url: 'http://localhost:8080',
        reference: ''
    }) {
        await (0, pkg_1.default)();
        let url = engine_opts.url ?? 'http://localhost:8080';
        // trim trailing slash
        url = url.replace(/\/$/, '');
        url = `${url}/internal/v1/evaluation/snapshot/namespace/${namespace}`;
        if (engine_opts.reference) {
            url = `${url}?ref=${engine_opts.reference}`;
        }
        const headers = new Headers();
        headers.append('Accept', 'application/json');
        headers.append('Content-Type', 'application/json');
        if (engine_opts.authentication) {
            if ('client_token' in engine_opts.authentication) {
                headers.append('Authorization', `Bearer ${engine_opts.authentication.client_token}`);
            }
            else if ('jwt_token' in engine_opts.authentication) {
                headers.append('Authorization', `JWT ${engine_opts.authentication.jwt_token}`);
            }
        }
        const fetcher = async () => {
            const resp = await fetch(url, {
                method: 'GET',
                headers
            });
            if (!resp.ok) {
                throw new Error('Failed to fetch data');
            }
            return resp;
        };
        const resp = await fetcher();
        const data = await resp.json();
        const engine = new pkg_1.Engine(namespace, data);
        return new FliptEvaluationClient(engine, fetcher);
    }
    async refresh() {
        const resp = await this.fetcher();
        const data = await resp.json();
        this.engine.snapshot(data);
    }
    evaluateVariant(flag_key, entity_id, context) {
        const evaluation_request = {
            flag_key,
            entity_id,
            context
        };
        return this.engine.evaluate_variant(evaluation_request);
    }
    evaluateBoolean(flag_key, entity_id, context) {
        const evaluation_request = {
            flag_key,
            entity_id,
            context
        };
        return this.engine.evaluate_boolean(evaluation_request);
    }
}
exports.FliptEvaluationClient = FliptEvaluationClient;

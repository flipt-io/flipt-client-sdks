import { FliptEvaluationClient } from '.';
import { v4 as uuidv4 } from 'uuid';
import { BooleanResult } from './models';

describe('evaluation', () => {
  const fliptUrl = process.env['FLIPT_URL'];
  if (!fliptUrl) {
    console.error('please set the FLIPT_URL environment variable');
    process.exit(1);
  }

  const authToken = process.env['FLIPT_AUTH_TOKEN'];
  if (!authToken) {
    console.error('please set the FLIPT_AUTH_TOKEN environment variable');
    process.exit(1);
  }

  let client: FliptEvaluationClient;
  let entityId: string;
  let context: Record<string, any>;

  beforeAll(() => {
    client = new FliptEvaluationClient('e2e', {
      url: fliptUrl,
      authentication: {
        client_token: authToken
      }
    });
  });

  afterAll(() => {
    client.close();
  });

  beforeEach(() => {
    entityId = uuidv4();
    context = {};
  });

  describe('boolean-flag', () => {
    describe('simple', () => {
      describe('disabled', () => {
        const flagKey = 'prd0002-disabled';
        let observed: BooleanResult;

        beforeEach(() => {
          observed = client.evaluateBoolean(flagKey, entityId, context);
        });

        it('should return false', () => {
          expect(observed.status).toEqual('success');
          expect(observed.error_message).toBeNull();
          expect(observed.result!.enabled).toBe(false);
          expect(observed.result!.reason).toBe('MATCH_EVALUATION_REASON');
        });
      });
    });
  });
});

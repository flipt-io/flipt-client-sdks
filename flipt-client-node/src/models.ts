/**
 * Represents the options for a fetcher function.
 */
export interface IFetcherOptions {
  /**
   * ETag for the request.
   */
  etag?: string;
  /**
   * Abort signal for the request.
   */
  signal?: AbortSignal;
}

/**
 * Represents a function that fetches a response from a remote Flipt instance.
 */
export interface IFetcher {
  (url: string, opts?: IFetcherOptions): Promise<Response>;
}

/**
 * Authentication strategy for the client.
 */
export interface AuthenticationStrategy {}

/**
 * Client Token Authentication.
 * @see {@link https://docs.flipt.io/authentication/using-tokens}.
 */
export interface ClientTokenAuthentication extends AuthenticationStrategy {
  /**
   * Flipt client token.
   */
  clientToken: string;
}

/**
 * JWT Authentication.
 * @see {@link https://docs.flipt.io/authentication/using-jwts}
 */
export interface JWTAuthentication extends AuthenticationStrategy {
  /**
   * JWT token.
   */
  jwtToken: string;
}

export type FetchMode = 'polling' | 'streaming';

export interface ClientOptions<T extends AuthenticationStrategy> {
  /**
   * The URL of the upstream Flipt instance.
   *
   * @defaultValue `http://localhost:8080`
   */
  url?: string;
  /**
   * The interval (seconds) in which to fetch new flag state.
   * @defaultValue `120` seconds.
   */
  updateInterval?: number;
  /**
   * The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication.
   *
   * @remarks
   * Client supports the following authentication strategies: No Authentication, {@link ClientTokenAuthentication} and {@link JWTAuthentication}.
   */
  authentication?: T;
  /**
   * The reference to use when fetching flag state. If not provided, reference will not be used.
   * @see {@link https://docs.flipt.io/guides/user/using-references}
   */
  reference?: string;
  /**
   * The fetcher to use when fetching flag state. If not provided, the client will default to a default fetcher.
   */
  fetcher?: IFetcher;
  /**
   * The fetch mode to use when fetching flag state. If not provided, the client will default to polling.
   * @defaultValue `polling`
   *
   * @remarks
   * Note: Streaming is currently only supported when using the SDK with Flipt Cloud (https://flipt.io/cloud).
   */
  fetchMode?: FetchMode;
}

/**
 * Represents a request to evaluate a feature flag.
 */
export interface EvaluationRequest {
  /**
   * Feature flag key
   */
  flagKey: string;
  /**
   * Entity identifier
   */
  entityId: string;
  /**
   * Context information for flag evaluation
   *
   * @example
   * ```
   * {
   *  fizz: 'buzz',
   *  kind: 'small'
   * }
   * ```
   */
  context: object;
}

/**
 * Represents a feature flag in the system.
 */
export interface Flag {
  /**
   * Unique key for the flag.
   */
  key: string;
  /**
   * Status of the flag. For a Variant flag, this represents whether
   * the flag is enabled. For a Boolean flag, this represents the
   * default value.
   */
  enabled: boolean;
  /**
   * Type of the feature flag. This can be either 'BOOLEAN_FLAG_TYPE' or 'VARIANT_FLAG_TYPE'.
   */
  type: string;
}

/**
 * Represents the response returned after evaluating a feature flag variant.
 */
export interface VariantEvaluationResponse {
  /** Indicates whether the feature flag evaluation resulted in a match. */
  match: boolean;
  /** List of segment keys that were used to determine the match. */
  segmentKeys: string[];
  /** Reason for the result that occurred during the evaluation. */
  reason: string;
  /** Key of the feature flag that was being evaluated. */
  flagKey: string;
  /** Variant key that was returned if the evaluation resulted in a match. */
  variantKey: string;
  /** Additional data attached to the variant if the evaluation resulted in a match. */
  variantAttachment: string;
  /** Duration of the request in milliseconds. */
  requestDurationMillis: number;
  /** Timestamp when the response was generated. */
  timestamp: string;
}

/**
 * Represents the response returned after evaluating a feature flag.
 */
export interface BooleanEvaluationResponse {
  /** Evaluation value of the flag. */
  enabled: boolean;
  /** Key of the feature flag that was being evaluated. */
  flagKey: string;
  /** Reason for the result that occurred during the evaluation. */
  reason: string;
  /** Duration of the request in milliseconds. */
  requestDurationMillis: number;
  /** Timestamp when the response was generated. */
  timestamp: string;
}

/**
 * Represents the response returned when an error occurs during the evaluation of a feature flag.
 */
export interface ErrorEvaluationResponse {
  /** Key of the feature flag that was being evaluated. */
  flagKey: string;
  /** Key of the namespace in which the feature flag resides. */
  namespaceKey: string;
  /** Reason for the result that occurred during the evaluation. */
  reason: string;
}

/**
 * Represents a response object that encapsulates various types of evaluation results.
 */
export interface EvaluationResponse {
  /**
   * The type of evaluation response. Possible values include 'VARIANT_EVALUATION_RESPONSE_TYPE',
   * 'BOOLEAN_EVALUATION_RESPONSE_TYPE', or 'ERROR_EVALUATION_RESPONSE_TYPE'.
   */
  type: string;
  /** Boolean evaluation response base on the type */
  booleanEvaluationResponse?: BooleanEvaluationResponse;
  /** Variant evaluation response base on the type */
  variantEvaluationResponse?: VariantEvaluationResponse;
  /** Error evaluation response base on the type*/
  errorEvaluationResponse?: ErrorEvaluationResponse;
}

/**
 * Represents the response returned after batch evaluating multiple feature flags.
 */
export interface BatchEvaluationResponse {
  /** Array containing individual evaluation responses for each feature flag. */
  responses: EvaluationResponse[];
  /** Duration of the request in milliseconds. */
  requestDurationMillis: number;
}

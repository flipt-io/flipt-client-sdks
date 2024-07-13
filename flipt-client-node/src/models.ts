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
  client_token: string;
}

/**
 * JWT Authentication.
 * @see {@link https://docs.flipt.io/authentication/using-jwts}
 */
export interface JWTAuthentication extends AuthenticationStrategy {
  /**
   * JWT token.
   */
  jwt_token: string;
}

export interface EngineOpts<T> {
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
  update_interval?: number;
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
}

export interface EvaluationRequest {
  /**
   * Feature flag key
   */
  flag_key: string;
  /**
   * Entity identifier
   */
  entity_id: string;
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
  segment_keys: string[];
  /** Reason for the result that occurred during the evaluation. */
  reason: string;
  /** Key of the feature flag that was being evaluated. */
  flag_key: string;
  /** Variant key that was returned if the evaluation resulted in a match. */
  variant_key: string;
  /** Additional data attached to the variant if the evaluation resulted in a match. */
  variant_attachment: string;
  /** Duration of the request in milliseconds. */
  request_duration_millis: number;
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
  flag_key: string;
  /** Reason for the result that occurred during the evaluation. */
  reason: string;
  /** Duration of the request in milliseconds. */
  request_duration_millis: number;
  /** Timestamp when the response was generated. */
  timestamp: string;
}

/**
 * Represents the response returned when an error occurs during the evaluation of a feature flag.
 */
export interface ErrorEvaluationResponse {
  /** Key of the feature flag that was being evaluated. */
  flag_key: string;
  /** Key of the namespace in which the feature flag resides. */
  namespace_key: string;
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
  boolean_evaluation_response?: BooleanEvaluationResponse;
  /** Variant evaluation response base on the type */
  variant_evaluation_response?: VariantEvaluationResponse;
  /** Error evaluation response base on the type*/
  error_evaluation_response?: ErrorEvaluationResponse;
}

/**
 * Represents the response returned after batch evaluating multiple feature flags.
 */
export interface BatchEvaluationResponse {
  /** Array containing individual evaluation responses for each feature flag. */
  responses: EvaluationResponse[];
  /** Duration of the request in milliseconds. */
  request_duration_millis: number;
}

/**
 * Represents a specialized result object for variant evaluation, extending
 * the generic Result interface with a specific type VariantEvaluationResponse.
 */
export interface VariantResult extends Result<VariantEvaluationResponse> {}

/**
 * Represents a specialized result object for boolean evaluation, extending
 * the generic Result interface with a specific type BooleanEvaluationResponse.
 */
export interface BooleanResult extends Result<BooleanEvaluationResponse> {}

/**
 * Represents a specialized result object for batch evaluation, extending
 * the generic Result interface with a specific type BatchEvaluationResponse.
 */
export interface BatchResult extends Result<BatchEvaluationResponse> {}

export interface Result<T> {
  /** Status of the result - `success` or `failure`. */
  status: string;
  /** Actual result of type T if the operation was successful. */
  result?: T;
  /** Error message describing the reason for failure, if applicable.*/
  error_message: string;
}

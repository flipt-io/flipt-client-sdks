import 'dart:io';
import 'package:json_annotation/json_annotation.dart';
import 'package:flipt_client/src/errors.dart';

part 'models.g.dart';

enum FetchMode {
  @JsonValue('polling')
  polling,
  @JsonValue('streaming')
  streaming,
}

enum ErrorStrategy {
  @JsonValue('fail')
  fail,
  @JsonValue('fallback')
  fallback,
}

/// Configuration for TLS connections to Flipt servers.
/// Provides options for custom CA certificates, client certificates for mutual TLS,
/// and insecure mode for development.
@JsonSerializable()
class TlsConfig {
  /// Path to the CA certificate file (PEM format).
  final String? caCertFile;

  /// CA certificate content (PEM format).
  /// Takes precedence over caCertFile if both are provided.
  final String? caCertData;

  /// Whether to skip certificate verification.
  /// WARNING: Only use this in development environments!
  final bool? insecureSkipVerify;

  /// Path to the client certificate file for mutual TLS (PEM format).
  final String? clientCertFile;

  /// Path to the client private key file for mutual TLS (PEM format).
  final String? clientKeyFile;

  /// Client certificate content for mutual TLS (PEM format).
  /// Takes precedence over clientCertFile if both are provided.
  final String? clientCertData;

  /// Client private key content for mutual TLS (PEM format).
  /// Takes precedence over clientKeyFile if both are provided.
  final String? clientKeyData;

  TlsConfig({
    this.caCertFile,
    this.caCertData,
    this.insecureSkipVerify,
    this.clientCertFile,
    this.clientKeyFile,
    this.clientCertData,
    this.clientKeyData,
  }) {
    _validateFiles();
  }

  /// Creates a TLS configuration for insecure connections (development only).
  /// WARNING: Only use this in development environments!
  static TlsConfig insecure() {
    return TlsConfig(insecureSkipVerify: true);
  }

  /// Creates a TLS configuration with a custom CA certificate file.
  static TlsConfig withCaCertFile(String caCertFile) {
    if (caCertFile.isEmpty) {
      throw ValidationError('CA certificate file path cannot be empty');
    }
    return TlsConfig(caCertFile: caCertFile);
  }

  /// Creates a TLS configuration with CA certificate data.
  static TlsConfig withCaCertData(String caCertData) {
    if (caCertData.isEmpty) {
      throw ValidationError('CA certificate data cannot be empty');
    }
    return TlsConfig(caCertData: caCertData);
  }

  /// Creates a TLS configuration for mutual TLS using certificate files.
  static TlsConfig withMutualTls(String clientCertFile, String clientKeyFile) {
    if (clientCertFile.isEmpty) {
      throw ValidationError('Client certificate file path cannot be empty');
    }
    if (clientKeyFile.isEmpty) {
      throw ValidationError('Client key file path cannot be empty');
    }
    return TlsConfig(
      clientCertFile: clientCertFile,
      clientKeyFile: clientKeyFile,
    );
  }

  /// Creates a TLS configuration for mutual TLS using certificate data.
  static TlsConfig withMutualTlsData(
      String clientCertData, String clientKeyData) {
    if (clientCertData.isEmpty) {
      throw ValidationError('Client certificate data cannot be empty');
    }
    if (clientKeyData.isEmpty) {
      throw ValidationError('Client key data cannot be empty');
    }
    return TlsConfig(
      clientCertData: clientCertData,
      clientKeyData: clientKeyData,
    );
  }

  void _validateFiles() {
    if (caCertFile != null && caCertFile!.isNotEmpty) {
      if (!File(caCertFile!).existsSync()) {
        throw ValidationError(
            'CA certificate file does not exist: $caCertFile');
      }
    }
    if (clientCertFile != null && clientCertFile!.isNotEmpty) {
      if (!File(clientCertFile!).existsSync()) {
        throw ValidationError(
            'Client certificate file does not exist: $clientCertFile');
      }
    }
    if (clientKeyFile != null && clientKeyFile!.isNotEmpty) {
      if (!File(clientKeyFile!).existsSync()) {
        throw ValidationError('Client key file does not exist: $clientKeyFile');
      }
    }
  }

  factory TlsConfig.fromJson(Map<String, dynamic> json) =>
      _$TlsConfigFromJson(json);
  Map<String, dynamic> toJson() => _$TlsConfigToJson(this);
}

/// Options for configuring the Flipt client.
@JsonSerializable()
class Options {
  /// The environment to use (default: 'default').
  final String? environment;

  /// The namespace to use (default: 'default').
  final String? namespace;

  /// The Flipt server URL (default: 'http://localhost:8080').
  final String? url;

  /// The reference to use for fetching flag state.
  final String? reference;

  /// The request timeout duration.
  final Duration? requestTimeout;

  /// The update interval duration (default: 120 seconds).
  final Duration? updateInterval;

  /// Authentication options (client_token or jwt_token).
  final Map<String, dynamic>? authentication;

  /// The initial snapshot to use for the client.
  final String? snapshot;

  /// The fetch mode to use (polling or streaming).
  /// Note: Streaming is only supported when using the SDK with Flipt Cloud (https://flipt.io/cloud) or Flipt v2 (https://docs.flipt.io/v2)
  final FetchMode? fetchMode;

  /// The error strategy to use (fail or fallback).
  final ErrorStrategy? errorStrategy;

  /// TLS configuration for connecting to servers with custom certificates.
  final TlsConfig? tlsConfig;

  Options({
    this.url = 'http://localhost:8080',
    this.environment = 'default',
    this.namespace = 'default',
    this.reference,
    this.requestTimeout,
    this.updateInterval = const Duration(seconds: 120),
    this.authentication,
    this.fetchMode = FetchMode.polling,
    this.errorStrategy = ErrorStrategy.fail,
    this.snapshot,
    this.tlsConfig,
  }) {
    if (url != null && url!.isEmpty) {
      throw ValidationError('url must not be empty');
    }
    if (environment != null && environment!.isEmpty) {
      throw ValidationError('environment must not be empty');
    }
    if (namespace != null && namespace!.isEmpty) {
      throw ValidationError('namespace must not be empty');
    }
    if (requestTimeout != null && requestTimeout!.isNegative) {
      throw ValidationError('requestTimeout must not be negative');
    }
    if (updateInterval != null && updateInterval!.isNegative) {
      throw ValidationError('updateInterval must not be negative');
    }
  }

  factory Options.fromJson(Map<String, dynamic> json) => Options(
        url: json['url'] as String? ?? 'http://localhost:8080',
        environment: json['environment'] as String? ?? 'default',
        namespace: json['namespace'] as String? ?? 'default',
        reference: json['reference'] as String?,
        requestTimeout: json['request_timeout'] != null
            ? Duration(seconds: json['request_timeout'] as int)
            : null,
        updateInterval: json['update_interval'] != null
            ? Duration(seconds: json['update_interval'] as int)
            : const Duration(seconds: 120),
        authentication: json['authentication'] as Map<String, dynamic>?,
        fetchMode:
            $enumDecodeNullable(_$FetchModeEnumMap, json['fetch_mode']) ??
                FetchMode.polling,
        errorStrategy: $enumDecodeNullable(
                _$ErrorStrategyEnumMap, json['error_strategy']) ??
            ErrorStrategy.fail,
        snapshot: json['snapshot'] as String?,
        tlsConfig: json['tls_config'] != null
            ? TlsConfig.fromJson(json['tls_config'] as Map<String, dynamic>)
            : null,
      );
  Map<String, dynamic> toJson() => {
        'url': url,
        'environment': environment,
        'namespace': namespace,
        'reference': reference,
        'request_timeout': requestTimeout?.inSeconds,
        'update_interval': updateInterval?.inSeconds,
        'authentication': authentication,
        'fetch_mode': fetchMode != null ? _$FetchModeEnumMap[fetchMode] : null,
        'error_strategy': errorStrategy != null
            ? _$ErrorStrategyEnumMap[errorStrategy]
            : null,
        'snapshot': snapshot,
        'tls_config': tlsConfig?.toJson(),
      };

  static Options withClientToken(
    String token, {
    String? environment,
    String? namespace,
    String? url,
    String? reference,
    Duration? requestTimeout,
    Duration? updateInterval,
    FetchMode? fetchMode,
    ErrorStrategy? errorStrategy,
    TlsConfig? tlsConfig,
  }) {
    return Options(
      url: url,
      environment: environment,
      namespace: namespace,
      reference: reference,
      requestTimeout: requestTimeout,
      updateInterval: updateInterval,
      authentication: {
        'client_token': token,
      },
      fetchMode: fetchMode,
      errorStrategy: errorStrategy,
      tlsConfig: tlsConfig,
    );
  }

  static Options withJwtToken(
    String token, {
    String? url,
    String? environment,
    String? namespace,
    String? reference,
    Duration? requestTimeout,
    Duration? updateInterval,
    FetchMode? fetchMode,
    ErrorStrategy? errorStrategy,
    TlsConfig? tlsConfig,
  }) {
    return Options(
      url: url,
      environment: environment,
      namespace: namespace,
      reference: reference,
      requestTimeout: requestTimeout,
      updateInterval: updateInterval,
      authentication: {
        'jwt_token': token,
      },
      fetchMode: fetchMode,
      errorStrategy: errorStrategy,
      tlsConfig: tlsConfig,
    );
  }
}

/// Represents a flag in Flipt.
@JsonSerializable()
class Flag {
  /// The key of the flag.
  final String key;

  /// Whether the flag is enabled.
  final bool enabled;

  /// The type of the flag.
  final String type;

  Flag({
    required this.key,
    required this.enabled,
    required this.type,
  });

  factory Flag.fromJson(Map<String, dynamic> json) => _$FlagFromJson(json);
  Map<String, dynamic> toJson() => _$FlagToJson(this);
}

/// Request for evaluating a flag.
@JsonSerializable()
class EvaluationRequest {
  /// The key of the flag to evaluate.
  final String flagKey;

  /// The entity ID for evaluation.
  final String entityId;

  /// The evaluation context.
  final Map<String, dynamic> context;

  EvaluationRequest({
    required this.flagKey,
    required this.entityId,
    required this.context,
  });

  factory EvaluationRequest.fromJson(Map<String, dynamic> json) =>
      _$EvaluationRequestFromJson(json);
  Map<String, dynamic> toJson() => _$EvaluationRequestToJson(this);
}

enum Status {
  @JsonValue('success')
  success,
  @JsonValue('failure')
  failure,
}

/// Generic result wrapper for Flipt responses.
@JsonSerializable(genericArgumentFactories: true)
class Result<T> {
  /// The status of the response.
  final Status status;

  /// The result object, if successful.
  final T? result;

  /// The error message, if any.
  final String? errorMessage;

  Result({
    required this.status,
    this.result,
    this.errorMessage,
  });

  factory Result.fromJson(
          Map<String, dynamic> json, T Function(Object?) fromJsonT) =>
      _$ResultFromJson(
          json, (object) => fromJsonT(object as Map<String, dynamic>));
}

/// Response for a variant flag evaluation.
@JsonSerializable()
class VariantEvaluationResponse {
  /// Whether the evaluation matched.
  final bool match;

  /// The segment keys involved in the evaluation.
  final List<String> segmentKeys;

  /// The reason for the evaluation result.
  final String reason;

  /// The flag key.
  final String flagKey;

  /// The variant key returned.
  final String variantKey;

  /// The variant attachment, if any.
  final String variantAttachment;

  /// The request duration in milliseconds.
  final double requestDurationMillis;

  /// The timestamp of the evaluation.
  final String timestamp;

  VariantEvaluationResponse({
    required this.match,
    required this.segmentKeys,
    required this.reason,
    required this.flagKey,
    required this.variantKey,
    required this.variantAttachment,
    required this.requestDurationMillis,
    required this.timestamp,
  });

  factory VariantEvaluationResponse.fromJson(Map<String, dynamic> json) =>
      _$VariantEvaluationResponseFromJson(json);
  Map<String, dynamic> toJson() => _$VariantEvaluationResponseToJson(this);
}

/// Response for a boolean flag evaluation.
@JsonSerializable()
class BooleanEvaluationResponse {
  /// Whether the flag is enabled.
  final bool enabled;

  /// The flag key.
  final String flagKey;

  /// The reason for the evaluation result.
  final String reason;

  /// The request duration in milliseconds.
  final double requestDurationMillis;

  /// The timestamp of the evaluation.
  final String timestamp;

  BooleanEvaluationResponse({
    required this.enabled,
    required this.flagKey,
    required this.reason,
    required this.requestDurationMillis,
    required this.timestamp,
  });

  factory BooleanEvaluationResponse.fromJson(Map<String, dynamic> json) =>
      _$BooleanEvaluationResponseFromJson(json);
  Map<String, dynamic> toJson() => _$BooleanEvaluationResponseToJson(this);
}

/// Response for an error during evaluation.
@JsonSerializable()
class ErrorEvaluationResponse {
  /// The flag key for which the error occurred.
  final String flagKey;

  /// The namespace key for which the error occurred.
  final String namespaceKey;

  /// The reason for the error.
  final String reason;

  ErrorEvaluationResponse({
    required this.flagKey,
    required this.namespaceKey,
    required this.reason,
  });

  factory ErrorEvaluationResponse.fromJson(Map<String, dynamic> json) =>
      _$ErrorEvaluationResponseFromJson(json);
  Map<String, dynamic> toJson() => _$ErrorEvaluationResponseToJson(this);
}

/// Response for a single evaluation (variant, boolean, or error).
@JsonSerializable()
class EvaluationResponse {
  /// The type of the response.
  final String type;

  /// The boolean evaluation response, if applicable.
  final BooleanEvaluationResponse? booleanEvaluationResponse;

  /// The variant evaluation response, if applicable.
  final VariantEvaluationResponse? variantEvaluationResponse;

  /// The error evaluation response, if applicable.
  final ErrorEvaluationResponse? errorEvaluationResponse;

  EvaluationResponse({
    required this.type,
    this.booleanEvaluationResponse,
    this.variantEvaluationResponse,
    this.errorEvaluationResponse,
  });

  factory EvaluationResponse.fromJson(Map<String, dynamic> json) =>
      _$EvaluationResponseFromJson(json);
  Map<String, dynamic> toJson() => _$EvaluationResponseToJson(this);
}

/// Response for a batch evaluation.
@JsonSerializable()
class BatchEvaluationResponse {
  /// The list of evaluation responses.
  final List<EvaluationResponse> responses;

  /// The request duration in milliseconds.
  final double requestDurationMillis;

  BatchEvaluationResponse({
    required this.responses,
    required this.requestDurationMillis,
  });

  factory BatchEvaluationResponse.fromJson(Map<String, dynamic> json) {
    return BatchEvaluationResponse(
      responses: (json['responses'] as List?)
              ?.map(
                  (e) => EvaluationResponse.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      requestDurationMillis: json['request_duration_millis'] as double? ?? 0.0,
    );
  }

  Map<String, dynamic> toJson() => {
        'responses': responses.map((e) => e.toJson()).toList(),
        'request_duration_millis': requestDurationMillis,
      };
}

/// Response for listing flags.
@JsonSerializable()
class FlagListResponse {
  /// The status of the response.
  final Status status;

  /// The list of flags, if successful.
  final List<Flag>? result;

  /// The error message, if any.
  final String? errorMessage;

  FlagListResponse({
    required this.status,
    required this.result,
    this.errorMessage,
  });

  factory FlagListResponse.fromJson(Map<String, dynamic> json) =>
      _$FlagListResponseFromJson(json);
  Map<String, dynamic> toJson() => _$FlagListResponseToJson(this);

  Result<List<Flag>> toResult() {
    return Result<List<Flag>>(
      status: status,
      result: result,
      errorMessage: errorMessage,
    );
  }
}

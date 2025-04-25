import 'package:json_annotation/json_annotation.dart';

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

/// Options for the Flipt client
@JsonSerializable()
class Options {
  final String? url;
  final String? reference;
  final int? requestTimeout;
  final int? updateInterval;
  final Map<String, dynamic>? authentication;

  /// Note: Streaming is currently only supported when using the SDK with Flipt Cloud (https://flipt.io/cloud).
  final FetchMode? fetchMode;
  final ErrorStrategy? errorStrategy;

  Options({
    this.url = 'http://localhost:8080',
    this.reference,
    this.requestTimeout,
    this.updateInterval = 120,
    this.authentication,
    this.fetchMode = FetchMode.polling,
    this.errorStrategy = ErrorStrategy.fail,
  });

  factory Options.fromJson(Map<String, dynamic> json) =>
      _$OptionsFromJson(json);
  Map<String, dynamic> toJson() => _$OptionsToJson(this);

  static Options withClientToken(
    String token, {
    String? url,
    String? reference,
    int? requestTimeout,
    int? updateInterval,
    FetchMode? fetchMode,
    ErrorStrategy? errorStrategy,
  }) {
    return Options(
      url: url,
      reference: reference,
      requestTimeout: requestTimeout,
      updateInterval: updateInterval,
      authentication: {
        'client_token': token,
      },
      fetchMode: fetchMode,
      errorStrategy: errorStrategy,
    );
  }

  static Options withJwtToken(
    String token, {
    String? url,
    String? reference,
    int? requestTimeout,
    int? updateInterval,
    FetchMode? fetchMode,
    ErrorStrategy? errorStrategy,
  }) {
    return Options(
      url: url,
      reference: reference,
      requestTimeout: requestTimeout,
      updateInterval: updateInterval,
      authentication: {
        'jwt_token': token,
      },
      fetchMode: fetchMode,
      errorStrategy: errorStrategy,
    );
  }
}

@JsonSerializable()
class Flag {
  final String key;
  final bool enabled;
  final String type;

  Flag({
    required this.key,
    required this.enabled,
    required this.type,
  });

  factory Flag.fromJson(Map<String, dynamic> json) => _$FlagFromJson(json);
  Map<String, dynamic> toJson() => _$FlagToJson(this);
}

@JsonSerializable()
class EvaluationRequest {
  final String flagKey;
  final String entityId;
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

@JsonSerializable(genericArgumentFactories: true)
class Result<T> {
  final Status status;
  final T? result;
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

@JsonSerializable()
class VariantEvaluationResponse {
  final bool match;
  final List<String> segmentKeys;
  final String reason;
  final String flagKey;
  final String variantKey;
  final String variantAttachment;
  final double requestDurationMillis;
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

@JsonSerializable()
class BooleanEvaluationResponse {
  final bool enabled;
  final String flagKey;
  final String reason;
  final double requestDurationMillis;
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

@JsonSerializable()
class ErrorEvaluationResponse {
  final String flagKey;
  final String namespaceKey;
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

@JsonSerializable()
class EvaluationResponse {
  final String type;
  final BooleanEvaluationResponse? booleanEvaluationResponse;
  final VariantEvaluationResponse? variantEvaluationResponse;
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

@JsonSerializable()
class BatchEvaluationResponse {
  final List<EvaluationResponse> responses;
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

@JsonSerializable()
class FlagListResponse {
  final Status status;
  final List<Flag>? result;
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

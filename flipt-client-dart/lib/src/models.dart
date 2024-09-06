import 'package:json_annotation/json_annotation.dart';

part 'models.g.dart';

@JsonSerializable()
class Options {
  final String? url;
  final String? reference;
  final int? updateInterval;
  final Map<String, dynamic>? authentication;

  Options({
    this.url = 'http://localhost:8080',
    this.reference,
    this.updateInterval = 120,
    this.authentication,
  });

  factory Options.fromJson(Map<String, dynamic> json) =>
      _$OptionsFromJson(json);
  Map<String, dynamic> toJson() => _$OptionsToJson(this);

  static Options withClientToken(
    String token, {
    String? url,
    String? reference,
    int? updateInterval,
  }) {
    return Options(
      url: url,
      reference: reference,
      updateInterval: updateInterval,
      authentication: {
        'client_token': token,
      },
    );
  }

  static Options withJwtToken(
    String token, {
    String? url,
    String? reference,
    int? updateInterval,
  }) {
    return Options(
      url: url,
      reference: reference,
      updateInterval: updateInterval,
      authentication: {
        'jwt_token': token,
      },
    );
  }
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

enum EvaluationStatus {
  @JsonValue('success')
  success,
  @JsonValue('failure')
  failure,
}

@JsonSerializable(genericArgumentFactories: true)
class EvaluationResponse<T> {
  final EvaluationStatus status;
  final T? result;
  final String? errorMessage;

  EvaluationResponse({
    required this.status,
    this.result,
    this.errorMessage,
  });

  factory EvaluationResponse.fromJson(
          Map<String, dynamic> json, T Function(Object?) fromJsonT) =>
      _$EvaluationResponseFromJson(
          json, (object) => fromJsonT(object as Map<String, dynamic>));
}

@JsonSerializable()
class VariantResult {
  final bool match;
  final List<String> segmentKeys;
  final String reason;
  final String flagKey;
  final String variantKey;
  final String variantAttachment;
  final double requestDurationMillis;
  final String timestamp;

  VariantResult({
    required this.match,
    required this.segmentKeys,
    required this.reason,
    required this.flagKey,
    required this.variantKey,
    required this.variantAttachment,
    required this.requestDurationMillis,
    required this.timestamp,
  });

  factory VariantResult.fromJson(Map<String, dynamic> json) =>
      _$VariantResultFromJson(json);
  Map<String, dynamic> toJson() => _$VariantResultToJson(this);
}

@JsonSerializable()
class BooleanResult {
  final bool enabled;
  final String flagKey;
  final String reason;
  final double requestDurationMillis;
  final String timestamp;

  BooleanResult({
    required this.enabled,
    required this.flagKey,
    required this.reason,
    required this.requestDurationMillis,
    required this.timestamp,
  });

  factory BooleanResult.fromJson(Map<String, dynamic> json) =>
      _$BooleanResultFromJson(json);
  Map<String, dynamic> toJson() => _$BooleanResultToJson(this);
}

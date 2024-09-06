// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Options _$OptionsFromJson(Map<String, dynamic> json) => Options(
      url: json['url'] as String? ?? 'http://localhost:8080',
      reference: json['reference'] as String?,
      updateInterval: (json['update_interval'] as num?)?.toInt() ?? 120,
      authentication: json['authentication'] as Map<String, dynamic>?,
    );

Map<String, dynamic> _$OptionsToJson(Options instance) => <String, dynamic>{
      'url': instance.url,
      'reference': instance.reference,
      'update_interval': instance.updateInterval,
      'authentication': instance.authentication,
    };

EvaluationRequest _$EvaluationRequestFromJson(Map<String, dynamic> json) =>
    EvaluationRequest(
      flagKey: json['flag_key'] as String,
      entityId: json['entity_id'] as String,
      context: json['context'] as Map<String, dynamic>,
    );

Map<String, dynamic> _$EvaluationRequestToJson(EvaluationRequest instance) =>
    <String, dynamic>{
      'flag_key': instance.flagKey,
      'entity_id': instance.entityId,
      'context': instance.context,
    };

EvaluationResponse<T> _$EvaluationResponseFromJson<T>(
  Map<String, dynamic> json,
  T Function(Object? json) fromJsonT,
) =>
    EvaluationResponse<T>(
      status: $enumDecode(_$EvaluationStatusEnumMap, json['status']),
      result: _$nullableGenericFromJson(json['result'], fromJsonT),
      errorMessage: json['error_message'] as String?,
    );

Map<String, dynamic> _$EvaluationResponseToJson<T>(
  EvaluationResponse<T> instance,
  Object? Function(T value) toJsonT,
) =>
    <String, dynamic>{
      'status': _$EvaluationStatusEnumMap[instance.status]!,
      'result': _$nullableGenericToJson(instance.result, toJsonT),
      'error_message': instance.errorMessage,
    };

const _$EvaluationStatusEnumMap = {
  EvaluationStatus.success: 'success',
  EvaluationStatus.failure: 'failure',
};

T? _$nullableGenericFromJson<T>(
  Object? input,
  T Function(Object? json) fromJson,
) =>
    input == null ? null : fromJson(input);

Object? _$nullableGenericToJson<T>(
  T? input,
  Object? Function(T value) toJson,
) =>
    input == null ? null : toJson(input);

VariantResult _$VariantResultFromJson(Map<String, dynamic> json) =>
    VariantResult(
      match: json['match'] as bool,
      segmentKeys: (json['segment_keys'] as List<dynamic>)
          .map((e) => e as String)
          .toList(),
      reason: json['reason'] as String,
      flagKey: json['flag_key'] as String,
      variantKey: json['variant_key'] as String,
      variantAttachment: json['variant_attachment'] as String,
      requestDurationMillis:
          (json['request_duration_millis'] as num).toDouble(),
      timestamp: json['timestamp'] as String,
    );

Map<String, dynamic> _$VariantResultToJson(VariantResult instance) =>
    <String, dynamic>{
      'match': instance.match,
      'segment_keys': instance.segmentKeys,
      'reason': instance.reason,
      'flag_key': instance.flagKey,
      'variant_key': instance.variantKey,
      'variant_attachment': instance.variantAttachment,
      'request_duration_millis': instance.requestDurationMillis,
      'timestamp': instance.timestamp,
    };

BooleanResult _$BooleanResultFromJson(Map<String, dynamic> json) =>
    BooleanResult(
      enabled: json['enabled'] as bool,
      flagKey: json['flag_key'] as String,
      reason: json['reason'] as String,
      requestDurationMillis:
          (json['request_duration_millis'] as num).toDouble(),
      timestamp: json['timestamp'] as String,
    );

Map<String, dynamic> _$BooleanResultToJson(BooleanResult instance) =>
    <String, dynamic>{
      'enabled': instance.enabled,
      'flag_key': instance.flagKey,
      'reason': instance.reason,
      'request_duration_millis': instance.requestDurationMillis,
      'timestamp': instance.timestamp,
    };

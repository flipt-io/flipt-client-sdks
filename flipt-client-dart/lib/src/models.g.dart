// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

TlsConfig _$TlsConfigFromJson(Map<String, dynamic> json) => TlsConfig(
      caCertFile: json['ca_cert_file'] as String?,
      caCertData: json['ca_cert_data'] as String?,
      insecureSkipVerify: json['insecure_skip_verify'] as bool?,
      clientCertFile: json['client_cert_file'] as String?,
      clientKeyFile: json['client_key_file'] as String?,
      clientCertData: json['client_cert_data'] as String?,
      clientKeyData: json['client_key_data'] as String?,
    );

Map<String, dynamic> _$TlsConfigToJson(TlsConfig instance) => <String, dynamic>{
      'ca_cert_file': instance.caCertFile,
      'ca_cert_data': instance.caCertData,
      'insecure_skip_verify': instance.insecureSkipVerify,
      'client_cert_file': instance.clientCertFile,
      'client_key_file': instance.clientKeyFile,
      'client_cert_data': instance.clientCertData,
      'client_key_data': instance.clientKeyData,
    };

Options _$OptionsFromJson(Map<String, dynamic> json) => Options(
      url: json['url'] as String? ?? 'http://localhost:8080',
      environment: json['environment'] as String? ?? 'default',
      namespace: json['namespace'] as String? ?? 'default',
      reference: json['reference'] as String?,
      requestTimeout: json['request_timeout'] == null
          ? null
          : Duration(microseconds: (json['request_timeout'] as num).toInt()),
      updateInterval: json['update_interval'] == null
          ? const Duration(seconds: 120)
          : Duration(microseconds: (json['update_interval'] as num).toInt()),
      authentication: json['authentication'] as Map<String, dynamic>?,
      fetchMode: $enumDecodeNullable(_$FetchModeEnumMap, json['fetch_mode']) ??
          FetchMode.polling,
      errorStrategy:
          $enumDecodeNullable(_$ErrorStrategyEnumMap, json['error_strategy']) ??
              ErrorStrategy.fail,
      snapshot: json['snapshot'] as String?,
      tlsConfig: json['tls_config'] == null
          ? null
          : TlsConfig.fromJson(json['tls_config'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$OptionsToJson(Options instance) => <String, dynamic>{
      'environment': instance.environment,
      'namespace': instance.namespace,
      'url': instance.url,
      'reference': instance.reference,
      'request_timeout': instance.requestTimeout?.inMicroseconds,
      'update_interval': instance.updateInterval?.inMicroseconds,
      'authentication': instance.authentication,
      'snapshot': instance.snapshot,
      'fetch_mode': _$FetchModeEnumMap[instance.fetchMode],
      'error_strategy': _$ErrorStrategyEnumMap[instance.errorStrategy],
      'tls_config': instance.tlsConfig?.toJson(),
    };

const _$FetchModeEnumMap = {
  FetchMode.polling: 'polling',
  FetchMode.streaming: 'streaming',
};

const _$ErrorStrategyEnumMap = {
  ErrorStrategy.fail: 'fail',
  ErrorStrategy.fallback: 'fallback',
};

Flag _$FlagFromJson(Map<String, dynamic> json) => Flag(
      key: json['key'] as String,
      enabled: json['enabled'] as bool,
      type: json['type'] as String,
    );

Map<String, dynamic> _$FlagToJson(Flag instance) => <String, dynamic>{
      'key': instance.key,
      'enabled': instance.enabled,
      'type': instance.type,
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

Result<T> _$ResultFromJson<T>(
  Map<String, dynamic> json,
  T Function(Object? json) fromJsonT,
) =>
    Result<T>(
      status: $enumDecode(_$StatusEnumMap, json['status']),
      result: _$nullableGenericFromJson(json['result'], fromJsonT),
      errorMessage: json['error_message'] as String?,
    );

Map<String, dynamic> _$ResultToJson<T>(
  Result<T> instance,
  Object? Function(T value) toJsonT,
) =>
    <String, dynamic>{
      'status': _$StatusEnumMap[instance.status]!,
      'result': _$nullableGenericToJson(instance.result, toJsonT),
      'error_message': instance.errorMessage,
    };

const _$StatusEnumMap = {
  Status.success: 'success',
  Status.failure: 'failure',
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

VariantEvaluationResponse _$VariantEvaluationResponseFromJson(
        Map<String, dynamic> json) =>
    VariantEvaluationResponse(
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

Map<String, dynamic> _$VariantEvaluationResponseToJson(
        VariantEvaluationResponse instance) =>
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

BooleanEvaluationResponse _$BooleanEvaluationResponseFromJson(
        Map<String, dynamic> json) =>
    BooleanEvaluationResponse(
      enabled: json['enabled'] as bool,
      flagKey: json['flag_key'] as String,
      reason: json['reason'] as String,
      requestDurationMillis:
          (json['request_duration_millis'] as num).toDouble(),
      timestamp: json['timestamp'] as String,
    );

Map<String, dynamic> _$BooleanEvaluationResponseToJson(
        BooleanEvaluationResponse instance) =>
    <String, dynamic>{
      'enabled': instance.enabled,
      'flag_key': instance.flagKey,
      'reason': instance.reason,
      'request_duration_millis': instance.requestDurationMillis,
      'timestamp': instance.timestamp,
    };

ErrorEvaluationResponse _$ErrorEvaluationResponseFromJson(
        Map<String, dynamic> json) =>
    ErrorEvaluationResponse(
      flagKey: json['flag_key'] as String,
      namespaceKey: json['namespace_key'] as String,
      reason: json['reason'] as String,
    );

Map<String, dynamic> _$ErrorEvaluationResponseToJson(
        ErrorEvaluationResponse instance) =>
    <String, dynamic>{
      'flag_key': instance.flagKey,
      'namespace_key': instance.namespaceKey,
      'reason': instance.reason,
    };

EvaluationResponse _$EvaluationResponseFromJson(Map<String, dynamic> json) =>
    EvaluationResponse(
      type: json['type'] as String,
      booleanEvaluationResponse: json['boolean_evaluation_response'] == null
          ? null
          : BooleanEvaluationResponse.fromJson(
              json['boolean_evaluation_response'] as Map<String, dynamic>),
      variantEvaluationResponse: json['variant_evaluation_response'] == null
          ? null
          : VariantEvaluationResponse.fromJson(
              json['variant_evaluation_response'] as Map<String, dynamic>),
      errorEvaluationResponse: json['error_evaluation_response'] == null
          ? null
          : ErrorEvaluationResponse.fromJson(
              json['error_evaluation_response'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$EvaluationResponseToJson(EvaluationResponse instance) =>
    <String, dynamic>{
      'type': instance.type,
      'boolean_evaluation_response':
          instance.booleanEvaluationResponse?.toJson(),
      'variant_evaluation_response':
          instance.variantEvaluationResponse?.toJson(),
      'error_evaluation_response': instance.errorEvaluationResponse?.toJson(),
    };

BatchEvaluationResponse _$BatchEvaluationResponseFromJson(
        Map<String, dynamic> json) =>
    BatchEvaluationResponse(
      responses: (json['responses'] as List<dynamic>)
          .map((e) => EvaluationResponse.fromJson(e as Map<String, dynamic>))
          .toList(),
      requestDurationMillis:
          (json['request_duration_millis'] as num).toDouble(),
    );

Map<String, dynamic> _$BatchEvaluationResponseToJson(
        BatchEvaluationResponse instance) =>
    <String, dynamic>{
      'responses': instance.responses.map((e) => e.toJson()).toList(),
      'request_duration_millis': instance.requestDurationMillis,
    };

FlagListResponse _$FlagListResponseFromJson(Map<String, dynamic> json) =>
    FlagListResponse(
      status: $enumDecode(_$StatusEnumMap, json['status']),
      result: (json['result'] as List<dynamic>?)
          ?.map((e) => Flag.fromJson(e as Map<String, dynamic>))
          .toList(),
      errorMessage: json['error_message'] as String?,
    );

Map<String, dynamic> _$FlagListResponseToJson(FlagListResponse instance) =>
    <String, dynamic>{
      'status': _$StatusEnumMap[instance.status]!,
      'result': instance.result?.map((e) => e.toJson()).toList(),
      'error_message': instance.errorMessage,
    };

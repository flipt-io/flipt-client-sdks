class EvaluationRequest {
  String? flagKey;
  String? entityId;
  Map<String, String>? context;

  Map toJson() {
    return {
      'flag_key': flagKey,
      'entity_id': entityId,
      'context': context,
    };
  }
}

abstract class Result<T> {
  final T result;
  final String? status;
  final String? errorMessage;

  Result({
    required this.result,
    this.status,
    this.errorMessage,
  });
}

class VariantResult extends Result<VariantEvaluationResponse> {
  VariantResult({
    required super.result,
    super.status,
    super.errorMessage,
  });

  factory VariantResult.fromJson(Map<String, dynamic> json) {
    return VariantResult(
      result: VariantEvaluationResponse.fromJson(json['result']),
      status: json['status'],
      errorMessage: json['error_message'],
    );
  }
}

class VariantEvaluationResponse {
  bool? match;
  List<String>? segmentKeys;
  String? reason;
  String? flagKey;
  String? variantKey;
  String? variantAttachment;
  double? requestDurationMillis;
  String? timestamp;

  VariantEvaluationResponse({
    this.match,
    this.segmentKeys,
    this.reason,
    this.flagKey,
    this.variantKey,
    this.variantAttachment,
    this.requestDurationMillis,
    this.timestamp,
  });

  factory VariantEvaluationResponse.fromJson(Map<String, dynamic> json) {
    return VariantEvaluationResponse(
      match: json['match'],
      segmentKeys: json['segment_keys']?.cast<String>(),
      reason: json['reason'],
      flagKey: json['flag_key'],
      variantKey: json['variant_key'],
      variantAttachment: json['variant_attachment'],
      requestDurationMillis: json['request_duration_millis'],
      timestamp: json['timestamp'],
    );
  }
}

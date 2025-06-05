/// Base class for all Flipt errors.
class FliptError implements Exception {
  final String message;
  FliptError(this.message);
  @override
  String toString() => 'FliptError: $message';
}

/// Thrown when a validation error occurs.
class ValidationError extends FliptError {
  ValidationError(String message) : super(message);
}

/// Thrown when an evaluation error occurs.
class EvaluationError extends FliptError {
  EvaluationError(String message) : super(message);
}

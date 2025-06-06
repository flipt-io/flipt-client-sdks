class FliptError(Exception):
    """Base exception for all Flipt client errors."""

    pass


class ValidationError(FliptError):
    """Raised when input validation fails."""

    pass


class EvaluationError(FliptError):
    """Raised when an evaluation fails (e.g., flag not found, engine error)."""

    pass

import pydantic
from packaging import version
from datetime import timedelta
import os

PYDANTIC_V2 = version.parse(pydantic.VERSION) >= version.parse("2.0.0")

from enum import Enum
from typing import List, Optional

from pydantic import BaseModel

if PYDANTIC_V2:
    from pydantic import RootModel


class EvaluationRequest(BaseModel):
    """Request for evaluating a flag for a given entity and context."""

    flag_key: str  #: The key of the flag to evaluate.
    entity_id: str  #: The entity identifier.
    context: dict  #: Additional context for evaluation.


class AuthenticationStrategy(BaseModel):
    """Base class for authentication strategies."""

    client_token: Optional[str] = None
    jwt_token: Optional[str] = None


class ClientTokenAuthentication(AuthenticationStrategy):
    """Client token authentication."""

    client_token: str


class JWTAuthentication(AuthenticationStrategy):
    """JWT authentication."""

    jwt_token: str


class FetchMode(Enum):
    """Flag fetch mode."""

    POLLING = "polling"
    STREAMING = "streaming"


class ErrorStrategy(Enum):
    """Error handling strategy."""

    FAIL = "fail"
    FALLBACK = "fallback"


class TlsConfig(BaseModel):
    """TLS configuration for connecting to Flipt servers with custom certificates."""

    ca_cert_file: Optional[str] = (
        None  #: Path to custom CA certificate file (PEM format).
    )
    ca_cert_data: Optional[str] = None  #: Raw CA certificate content (PEM format).
    insecure_skip_verify: Optional[bool] = (
        None  #: Skip certificate verification (development only).
    )
    insecure_skip_hostname_verify: Optional[bool] = (
        None  #: Skip hostname verification while maintaining certificate validation (development only).
    )
    client_cert_file: Optional[str] = (
        None  #: Client certificate file for mutual TLS (PEM format).
    )
    client_key_file: Optional[str] = (
        None  #: Client key file for mutual TLS (PEM format).
    )
    client_cert_data: Optional[str] = (
        None  #: Raw client certificate content (PEM format).
    )
    client_key_data: Optional[str] = None  #: Raw client key content (PEM format).

    if PYDANTIC_V2:
        from pydantic import field_validator

        @field_validator(
            "ca_cert_file", "client_cert_file", "client_key_file", mode="before"
        )
        @classmethod
        def validate_cert_files(cls, v, info):
            if v is not None and v.strip():
                if not os.path.exists(v):
                    field_name = (
                        info.field_name
                        if hasattr(info, "field_name")
                        else "certificate file"
                    )
                    raise ValueError(f"{field_name} does not exist: {v}")
            return v

    else:
        from pydantic import validator

        @validator("ca_cert_file", "client_cert_file", "client_key_file", pre=True)
        def validate_cert_files(cls, v, field):
            if v is not None and v.strip():
                if not os.path.exists(v):
                    field_name = (
                        field.name if hasattr(field, "name") else "certificate file"
                    )
                    raise ValueError(f"{field_name} does not exist: {v}")
            return v


class FlagType(str, Enum):
    """Type of flag."""

    BOOLEAN = "BOOLEAN_FLAG_TYPE"
    VARIANT = "VARIANT_FLAG_TYPE"


class ClientOptions(BaseModel):
    """Configuration options for FliptClient."""

    environment: Optional[str] = None  #: Environment name.
    namespace: Optional[str] = None  #: Namespace name.
    url: Optional[str] = None  #: Flipt server URL.
    request_timeout: Optional[timedelta] = None  #: Timeout for requests.
    update_interval: Optional[timedelta] = None  #: Polling/streaming update interval.
    authentication: Optional[AuthenticationStrategy] = None  #: Authentication strategy.
    reference: Optional[str] = None  #: Namespace or reference key.
    fetch_mode: Optional[FetchMode] = None  #: Fetch mode.
    error_strategy: Optional[ErrorStrategy] = None  #: Error handling strategy.
    snapshot: Optional[str] = None  #: Snapshot to use to initialize the client.
    tls_config: Optional[TlsConfig] = (
        None  #: TLS configuration for connecting to servers with custom certificates.
    )

    if PYDANTIC_V2:
        from pydantic import field_serializer, field_validator

        @field_serializer("request_timeout", "update_interval", mode="plain")
        def serialize_timedelta(value):
            if value is None:
                return None
            return int(value.total_seconds())

        @field_validator("request_timeout", "update_interval", mode="before")
        @classmethod
        def parse_timedelta(cls, v):
            if v is None or isinstance(v, timedelta):
                return v
            # Accept int/float as seconds
            if isinstance(v, (int, float)):
                return timedelta(seconds=v)
            raise TypeError(
                "Expected timedelta, int, or float for time interval fields."
            )

    else:
        from pydantic import validator

        @validator("request_timeout", "update_interval", pre=True)
        def parse_timedelta(cls, v):
            if v is None or isinstance(v, timedelta):
                return v
            if isinstance(v, (int, float)):
                return timedelta(seconds=v)
            raise TypeError(
                "Expected timedelta, int, or float for time interval fields."
            )

        def dict(self, *args, **kwargs):
            d = super().dict(*args, **kwargs)
            for k in ["request_timeout", "update_interval"]:
                v = d.get(k)
                if isinstance(v, timedelta):
                    d[k] = int(v.total_seconds())
            return d

        def json(self, *args, **kwargs):
            return super().json(*args, **kwargs)


class VariantEvaluationResponse(BaseModel):
    """Response for a variant flag evaluation."""

    match: bool  #: Whether the evaluation matched a segment.
    segment_keys: List[str]  #: List of matched segment keys.
    reason: str  #: Reason for the evaluation result.
    flag_key: str  #: The flag key.
    variant_key: str  #: The variant key returned.
    variant_attachment: Optional[str] = None  #: Optional variant attachment.
    request_duration_millis: float  #: Duration of the request in milliseconds.
    timestamp: str  #: Timestamp of the evaluation.


class BooleanEvaluationResponse(BaseModel):
    """Response for a boolean flag evaluation."""

    enabled: bool  #: Whether the flag is enabled.
    flag_key: str  #: The flag key.
    reason: str  #: Reason for the evaluation result.
    request_duration_millis: float  #: Duration of the request in milliseconds.
    timestamp: str  #: Timestamp of the evaluation.


class ErrorEvaluationResponse(BaseModel):
    """Response for an error during evaluation."""

    flag_key: str  #: The flag key.
    namespace_key: str  #: The namespace key.
    reason: str  #: Reason for the error.


class EvaluationResponse(BaseModel):
    """General evaluation response, can be boolean, variant, or error."""

    type: str  #: Type of the response.
    boolean_evaluation_response: Optional[BooleanEvaluationResponse] = None
    variant_evaluation_response: Optional[VariantEvaluationResponse] = None
    error_evaluation_response: Optional[ErrorEvaluationResponse] = None


class BatchEvaluationResponse(BaseModel):
    """Response for a batch evaluation request."""

    responses: List[EvaluationResponse]  #: List of evaluation responses.
    request_duration_millis: float  #: Duration of the batch request in milliseconds.


class VariantResult(BaseModel):
    """Result wrapper for variant evaluation."""

    status: str  #: Status of the evaluation (e.g., 'success', 'error').
    result: Optional[VariantEvaluationResponse] = (
        None  #: The evaluation response if successful.
    )
    error_message: Optional[str] = None  #: Error message if failed.


class BooleanResult(BaseModel):
    """Result wrapper for boolean evaluation."""

    status: str  #: Status of the evaluation (e.g., 'success', 'error').
    result: Optional[BooleanEvaluationResponse] = (
        None  #: The evaluation response if successful.
    )
    error_message: Optional[str] = None  #: Error message if failed.


class BatchResult(BaseModel):
    """Result wrapper for batch evaluation."""

    status: str  #: Status of the evaluation (e.g., 'success', 'error').
    result: Optional[BatchEvaluationResponse] = (
        None  #: The batch evaluation response if successful.
    )
    error_message: Optional[str] = None  #: Error message if failed.


class Flag(BaseModel):
    """Feature flag definition."""

    key: str  #: The flag key.
    enabled: bool  #: Whether the flag is enabled.
    type: FlagType  #: The type of the flag.
    description: Optional[str] = None  #: Optional description of the flag.


# --- FlagList root model compatibility ---
if PYDANTIC_V2:

    class FlagList(RootModel):
        """List of feature flags."""

        root: List[Flag]

        def __iter__(self):
            return iter(self.root)

        def __getitem__(self, item):
            return self.root[item]

        def __len__(self):
            return len(self.root)

else:

    class FlagList(BaseModel):
        """List of feature flags."""

        __root__: List[Flag]

        def __iter__(self):
            return iter(self.__root__)

        def __getitem__(self, item):
            return self.__root__[item]

        def __len__(self):
            return len(self.__root__)


class ListFlagsResult(BaseModel):
    """Result wrapper for listing flags."""

    status: str  #: Status of the operation.
    result: Optional[FlagList] = None  #: The list of flags if successful.
    error_message: Optional[str] = None  #: Error message if failed.


# --- Serialization/Deserialization shims ---
def model_to_json(model, *, exclude_none: bool = False):
    """Serialize a Pydantic model to JSON, compatible with v1 and v2."""
    if hasattr(model, "model_dump_json"):
        return model.model_dump_json(exclude_none=exclude_none)
    return model.json(exclude_none=exclude_none)


def model_from_json(cls, data):
    """Deserialize a Pydantic model from JSON, compatible with v1 and v2."""
    if hasattr(cls, "model_validate_json"):
        return cls.model_validate_json(data)
    return cls.parse_raw(data)

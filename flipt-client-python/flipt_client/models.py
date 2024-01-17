from pydantic import BaseModel
from typing import List, Optional


class EvaluationRequest(BaseModel):
    namespace_key: str
    flag_key: str
    entity_id: str
    context: dict


class EngineOpts(BaseModel):
    url: Optional[str] = None
    update_interval: Optional[int] = None
    auth_token: Optional[str] = None
    reference: Optional[str] = None


class VariantEvaluationResponse(BaseModel):
    match: bool
    segment_keys: List[str]
    reason: str
    flag_key: str
    variant_key: str
    variant_attachment: str
    request_duration_millis: float
    timestamp: str


class BooleanEvaluationResponse(BaseModel):
    enabled: bool
    flag_key: str
    reason: str
    request_duration_millis: float
    timestamp: str


class VariantResult(BaseModel):
    status: str
    result: Optional[VariantEvaluationResponse] = None
    error_message: Optional[str] = None


class BooleanResult(BaseModel):
    status: str
    result: Optional[BooleanEvaluationResponse] = None
    error_message: Optional[str] = None

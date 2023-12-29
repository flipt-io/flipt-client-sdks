#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationRequest {
    #[prost(string, tag = "1")]
    pub request_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub namespace_key: ::prost::alloc::string::String,
    #[prost(string, tag = "3")]
    pub flag_key: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub entity_id: ::prost::alloc::string::String,
    #[prost(map = "string, string", tag = "5")]
    pub context: ::std::collections::HashMap<
        ::prost::alloc::string::String,
        ::prost::alloc::string::String,
    >,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct BatchEvaluationRequest {
    #[prost(string, tag = "1")]
    pub request_id: ::prost::alloc::string::String,
    #[prost(message, repeated, tag = "3")]
    pub requests: ::prost::alloc::vec::Vec<EvaluationRequest>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct BatchEvaluationResponse {
    #[prost(string, tag = "1")]
    pub request_id: ::prost::alloc::string::String,
    #[prost(message, repeated, tag = "2")]
    pub responses: ::prost::alloc::vec::Vec<EvaluationResponse>,
    #[prost(double, tag = "3")]
    pub request_duration_millis: f64,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationResponse {
    #[prost(enumeration = "EvaluationResponseType", tag = "1")]
    pub r#type: i32,
    #[prost(oneof = "evaluation_response::Response", tags = "2, 3, 4")]
    pub response: ::core::option::Option<evaluation_response::Response>,
}
/// Nested message and enum types in `EvaluationResponse`.
pub mod evaluation_response {
    #[allow(clippy::derive_partial_eq_without_eq)]
    #[derive(Clone, PartialEq, ::prost::Oneof)]
    pub enum Response {
        #[prost(message, tag = "2")]
        BooleanResponse(super::BooleanEvaluationResponse),
        #[prost(message, tag = "3")]
        VariantResponse(super::VariantEvaluationResponse),
        #[prost(message, tag = "4")]
        ErrorResponse(super::ErrorEvaluationResponse),
    }
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct BooleanEvaluationResponse {
    #[prost(bool, tag = "1")]
    pub enabled: bool,
    #[prost(enumeration = "EvaluationReason", tag = "2")]
    pub reason: i32,
    #[prost(string, tag = "3")]
    pub request_id: ::prost::alloc::string::String,
    #[prost(double, tag = "4")]
    pub request_duration_millis: f64,
    #[prost(message, optional, tag = "5")]
    pub timestamp: ::core::option::Option<::pbjson_types::Timestamp>,
    #[prost(string, tag = "6")]
    pub flag_key: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct VariantEvaluationResponse {
    #[prost(bool, tag = "1")]
    pub r#match: bool,
    #[prost(string, repeated, tag = "2")]
    pub segment_keys: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
    #[prost(enumeration = "EvaluationReason", tag = "3")]
    pub reason: i32,
    #[prost(string, tag = "4")]
    pub variant_key: ::prost::alloc::string::String,
    #[prost(string, tag = "5")]
    pub variant_attachment: ::prost::alloc::string::String,
    #[prost(string, tag = "6")]
    pub request_id: ::prost::alloc::string::String,
    #[prost(double, tag = "7")]
    pub request_duration_millis: f64,
    #[prost(message, optional, tag = "8")]
    pub timestamp: ::core::option::Option<::pbjson_types::Timestamp>,
    #[prost(string, tag = "9")]
    pub flag_key: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct ErrorEvaluationResponse {
    #[prost(string, tag = "1")]
    pub flag_key: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub namespace_key: ::prost::alloc::string::String,
    #[prost(enumeration = "ErrorEvaluationReason", tag = "3")]
    pub reason: i32,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationDistribution {
    #[prost(string, tag = "1")]
    pub id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub rule_id: ::prost::alloc::string::String,
    #[prost(string, tag = "3")]
    pub variant_id: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub variant_key: ::prost::alloc::string::String,
    #[prost(string, tag = "5")]
    pub variant_attachment: ::prost::alloc::string::String,
    #[prost(float, tag = "6")]
    pub rollout: f32,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationRollout {
    #[prost(enumeration = "EvaluationRolloutType", tag = "1")]
    pub r#type: i32,
    #[prost(int32, tag = "2")]
    pub rank: i32,
    #[prost(oneof = "evaluation_rollout::Rule", tags = "3, 4")]
    pub rule: ::core::option::Option<evaluation_rollout::Rule>,
}
/// Nested message and enum types in `EvaluationRollout`.
pub mod evaluation_rollout {
    #[allow(clippy::derive_partial_eq_without_eq)]
    #[derive(Clone, PartialEq, ::prost::Oneof)]
    pub enum Rule {
        #[prost(message, tag = "3")]
        Segment(super::EvaluationRolloutSegment),
        #[prost(message, tag = "4")]
        Threshold(super::EvaluationRolloutThreshold),
    }
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationRolloutThreshold {
    #[prost(float, tag = "1")]
    pub percentage: f32,
    #[prost(bool, tag = "2")]
    pub value: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationRolloutSegment {
    #[prost(bool, tag = "1")]
    pub value: bool,
    #[prost(enumeration = "EvaluationSegmentOperator", tag = "2")]
    pub segment_operator: i32,
    #[prost(message, repeated, tag = "3")]
    pub segments: ::prost::alloc::vec::Vec<EvaluationSegment>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationSegment {
    #[prost(string, tag = "1")]
    pub key: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub name: ::prost::alloc::string::String,
    #[prost(string, tag = "3")]
    pub description: ::prost::alloc::string::String,
    #[prost(enumeration = "EvaluationSegmentMatchType", tag = "4")]
    pub match_type: i32,
    #[prost(message, optional, tag = "5")]
    pub created_at: ::core::option::Option<::pbjson_types::Timestamp>,
    #[prost(message, optional, tag = "6")]
    pub updated_at: ::core::option::Option<::pbjson_types::Timestamp>,
    #[prost(message, repeated, tag = "7")]
    pub constraints: ::prost::alloc::vec::Vec<EvaluationConstraint>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationFlag {
    #[prost(string, tag = "1")]
    pub key: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub name: ::prost::alloc::string::String,
    #[prost(string, tag = "3")]
    pub description: ::prost::alloc::string::String,
    #[prost(bool, tag = "4")]
    pub enabled: bool,
    #[prost(enumeration = "EvaluationFlagType", tag = "5")]
    pub r#type: i32,
    #[prost(message, optional, tag = "6")]
    pub created_at: ::core::option::Option<::pbjson_types::Timestamp>,
    #[prost(message, optional, tag = "7")]
    pub updated_at: ::core::option::Option<::pbjson_types::Timestamp>,
    #[prost(message, repeated, tag = "8")]
    pub rules: ::prost::alloc::vec::Vec<EvaluationRule>,
    #[prost(message, repeated, tag = "9")]
    pub rollouts: ::prost::alloc::vec::Vec<EvaluationRollout>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationConstraint {
    #[prost(string, tag = "1")]
    pub id: ::prost::alloc::string::String,
    #[prost(enumeration = "EvaluationConstraintComparisonType", tag = "2")]
    pub r#type: i32,
    #[prost(string, tag = "3")]
    pub property: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub operator: ::prost::alloc::string::String,
    #[prost(string, tag = "5")]
    pub value: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationRule {
    #[prost(string, tag = "1")]
    pub id: ::prost::alloc::string::String,
    #[prost(message, repeated, tag = "2")]
    pub segments: ::prost::alloc::vec::Vec<EvaluationSegment>,
    #[prost(int32, tag = "3")]
    pub rank: i32,
    #[prost(enumeration = "EvaluationSegmentOperator", tag = "4")]
    pub segment_operator: i32,
    #[prost(message, repeated, tag = "5")]
    pub distributions: ::prost::alloc::vec::Vec<EvaluationDistribution>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationNamespace {
    #[prost(string, tag = "1")]
    pub key: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationNamespaceSnapshot {
    #[prost(message, optional, tag = "1")]
    pub namespace: ::core::option::Option<EvaluationNamespace>,
    #[prost(message, repeated, tag = "2")]
    pub flags: ::prost::alloc::vec::Vec<EvaluationFlag>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EvaluationNamespaceSnapshotRequest {
    #[prost(string, tag = "1")]
    pub key: ::prost::alloc::string::String,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum EvaluationReason {
    UnknownEvaluationReason = 0,
    FlagDisabledEvaluationReason = 1,
    MatchEvaluationReason = 2,
    DefaultEvaluationReason = 3,
}
impl EvaluationReason {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            EvaluationReason::UnknownEvaluationReason => "UNKNOWN_EVALUATION_REASON",
            EvaluationReason::FlagDisabledEvaluationReason => {
                "FLAG_DISABLED_EVALUATION_REASON"
            }
            EvaluationReason::MatchEvaluationReason => "MATCH_EVALUATION_REASON",
            EvaluationReason::DefaultEvaluationReason => "DEFAULT_EVALUATION_REASON",
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "UNKNOWN_EVALUATION_REASON" => Some(Self::UnknownEvaluationReason),
            "FLAG_DISABLED_EVALUATION_REASON" => Some(Self::FlagDisabledEvaluationReason),
            "MATCH_EVALUATION_REASON" => Some(Self::MatchEvaluationReason),
            "DEFAULT_EVALUATION_REASON" => Some(Self::DefaultEvaluationReason),
            _ => None,
        }
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum ErrorEvaluationReason {
    UnknownErrorEvaluationReason = 0,
    NotFoundErrorEvaluationReason = 1,
}
impl ErrorEvaluationReason {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            ErrorEvaluationReason::UnknownErrorEvaluationReason => {
                "UNKNOWN_ERROR_EVALUATION_REASON"
            }
            ErrorEvaluationReason::NotFoundErrorEvaluationReason => {
                "NOT_FOUND_ERROR_EVALUATION_REASON"
            }
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "UNKNOWN_ERROR_EVALUATION_REASON" => Some(Self::UnknownErrorEvaluationReason),
            "NOT_FOUND_ERROR_EVALUATION_REASON" => {
                Some(Self::NotFoundErrorEvaluationReason)
            }
            _ => None,
        }
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum EvaluationResponseType {
    VariantEvaluationResponseType = 0,
    BooleanEvaluationResponseType = 1,
    ErrorEvaluationResponseType = 2,
}
impl EvaluationResponseType {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            EvaluationResponseType::VariantEvaluationResponseType => {
                "VARIANT_EVALUATION_RESPONSE_TYPE"
            }
            EvaluationResponseType::BooleanEvaluationResponseType => {
                "BOOLEAN_EVALUATION_RESPONSE_TYPE"
            }
            EvaluationResponseType::ErrorEvaluationResponseType => {
                "ERROR_EVALUATION_RESPONSE_TYPE"
            }
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "VARIANT_EVALUATION_RESPONSE_TYPE" => {
                Some(Self::VariantEvaluationResponseType)
            }
            "BOOLEAN_EVALUATION_RESPONSE_TYPE" => {
                Some(Self::BooleanEvaluationResponseType)
            }
            "ERROR_EVALUATION_RESPONSE_TYPE" => Some(Self::ErrorEvaluationResponseType),
            _ => None,
        }
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum EvaluationRolloutType {
    UnknownRolloutType = 0,
    SegmentRolloutType = 1,
    ThresholdRolloutType = 2,
}
impl EvaluationRolloutType {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            EvaluationRolloutType::UnknownRolloutType => "UNKNOWN_ROLLOUT_TYPE",
            EvaluationRolloutType::SegmentRolloutType => "SEGMENT_ROLLOUT_TYPE",
            EvaluationRolloutType::ThresholdRolloutType => "THRESHOLD_ROLLOUT_TYPE",
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "UNKNOWN_ROLLOUT_TYPE" => Some(Self::UnknownRolloutType),
            "SEGMENT_ROLLOUT_TYPE" => Some(Self::SegmentRolloutType),
            "THRESHOLD_ROLLOUT_TYPE" => Some(Self::ThresholdRolloutType),
            _ => None,
        }
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum EvaluationSegmentOperator {
    OrSegmentOperator = 0,
    AndSegmentOperator = 1,
}
impl EvaluationSegmentOperator {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            EvaluationSegmentOperator::OrSegmentOperator => "OR_SEGMENT_OPERATOR",
            EvaluationSegmentOperator::AndSegmentOperator => "AND_SEGMENT_OPERATOR",
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "OR_SEGMENT_OPERATOR" => Some(Self::OrSegmentOperator),
            "AND_SEGMENT_OPERATOR" => Some(Self::AndSegmentOperator),
            _ => None,
        }
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum EvaluationSegmentMatchType {
    AllSegmentMatchType = 0,
    AnySegmentMatchType = 1,
}
impl EvaluationSegmentMatchType {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            EvaluationSegmentMatchType::AllSegmentMatchType => "ALL_SEGMENT_MATCH_TYPE",
            EvaluationSegmentMatchType::AnySegmentMatchType => "ANY_SEGMENT_MATCH_TYPE",
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "ALL_SEGMENT_MATCH_TYPE" => Some(Self::AllSegmentMatchType),
            "ANY_SEGMENT_MATCH_TYPE" => Some(Self::AnySegmentMatchType),
            _ => None,
        }
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum EvaluationFlagType {
    VariantFlagType = 0,
    BooleanFlagType = 1,
}
impl EvaluationFlagType {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            EvaluationFlagType::VariantFlagType => "VARIANT_FLAG_TYPE",
            EvaluationFlagType::BooleanFlagType => "BOOLEAN_FLAG_TYPE",
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "VARIANT_FLAG_TYPE" => Some(Self::VariantFlagType),
            "BOOLEAN_FLAG_TYPE" => Some(Self::BooleanFlagType),
            _ => None,
        }
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord, ::prost::Enumeration)]
#[repr(i32)]
pub enum EvaluationConstraintComparisonType {
    UnknownConstraintComparisonType = 0,
    StringConstraintComparisonType = 1,
    NumberConstraintComparisonType = 2,
    BooleanConstraintComparisonType = 3,
    DatetimeConstraintComparisonType = 4,
}
impl EvaluationConstraintComparisonType {
    /// String value of the enum field names used in the ProtoBuf definition.
    ///
    /// The values are not transformed in any way and thus are considered stable
    /// (if the ProtoBuf definition does not change) and safe for programmatic use.
    pub fn as_str_name(&self) -> &'static str {
        match self {
            EvaluationConstraintComparisonType::UnknownConstraintComparisonType => {
                "UNKNOWN_CONSTRAINT_COMPARISON_TYPE"
            }
            EvaluationConstraintComparisonType::StringConstraintComparisonType => {
                "STRING_CONSTRAINT_COMPARISON_TYPE"
            }
            EvaluationConstraintComparisonType::NumberConstraintComparisonType => {
                "NUMBER_CONSTRAINT_COMPARISON_TYPE"
            }
            EvaluationConstraintComparisonType::BooleanConstraintComparisonType => {
                "BOOLEAN_CONSTRAINT_COMPARISON_TYPE"
            }
            EvaluationConstraintComparisonType::DatetimeConstraintComparisonType => {
                "DATETIME_CONSTRAINT_COMPARISON_TYPE"
            }
        }
    }
    /// Creates an enum from field names used in the ProtoBuf definition.
    pub fn from_str_name(value: &str) -> ::core::option::Option<Self> {
        match value {
            "UNKNOWN_CONSTRAINT_COMPARISON_TYPE" => {
                Some(Self::UnknownConstraintComparisonType)
            }
            "STRING_CONSTRAINT_COMPARISON_TYPE" => {
                Some(Self::StringConstraintComparisonType)
            }
            "NUMBER_CONSTRAINT_COMPARISON_TYPE" => {
                Some(Self::NumberConstraintComparisonType)
            }
            "BOOLEAN_CONSTRAINT_COMPARISON_TYPE" => {
                Some(Self::BooleanConstraintComparisonType)
            }
            "DATETIME_CONSTRAINT_COMPARISON_TYPE" => {
                Some(Self::DatetimeConstraintComparisonType)
            }
            _ => None,
        }
    }
}
/// Generated client implementations.
pub mod evaluation_service_client {
    #![allow(unused_variables, dead_code, missing_docs, clippy::let_unit_value)]
    use tonic::codegen::*;
    use tonic::codegen::http::Uri;
    #[derive(Debug, Clone)]
    pub struct EvaluationServiceClient<T> {
        inner: tonic::client::Grpc<T>,
    }
    impl EvaluationServiceClient<tonic::transport::Channel> {
        /// Attempt to create a new client by connecting to a given endpoint.
        pub async fn connect<D>(dst: D) -> Result<Self, tonic::transport::Error>
        where
            D: TryInto<tonic::transport::Endpoint>,
            D::Error: Into<StdError>,
        {
            let conn = tonic::transport::Endpoint::new(dst)?.connect().await?;
            Ok(Self::new(conn))
        }
    }
    impl<T> EvaluationServiceClient<T>
    where
        T: tonic::client::GrpcService<tonic::body::BoxBody>,
        T::Error: Into<StdError>,
        T::ResponseBody: Body<Data = Bytes> + Send + 'static,
        <T::ResponseBody as Body>::Error: Into<StdError> + Send,
    {
        pub fn new(inner: T) -> Self {
            let inner = tonic::client::Grpc::new(inner);
            Self { inner }
        }
        pub fn with_origin(inner: T, origin: Uri) -> Self {
            let inner = tonic::client::Grpc::with_origin(inner, origin);
            Self { inner }
        }
        pub fn with_interceptor<F>(
            inner: T,
            interceptor: F,
        ) -> EvaluationServiceClient<InterceptedService<T, F>>
        where
            F: tonic::service::Interceptor,
            T::ResponseBody: Default,
            T: tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
                Response = http::Response<
                    <T as tonic::client::GrpcService<tonic::body::BoxBody>>::ResponseBody,
                >,
            >,
            <T as tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
            >>::Error: Into<StdError> + Send + Sync,
        {
            EvaluationServiceClient::new(InterceptedService::new(inner, interceptor))
        }
        /// Compress requests with the given encoding.
        ///
        /// This requires the server to support it otherwise it might respond with an
        /// error.
        #[must_use]
        pub fn send_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.inner = self.inner.send_compressed(encoding);
            self
        }
        /// Enable decompressing responses.
        #[must_use]
        pub fn accept_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.inner = self.inner.accept_compressed(encoding);
            self
        }
        /// Limits the maximum size of a decoded message.
        ///
        /// Default: `4MB`
        #[must_use]
        pub fn max_decoding_message_size(mut self, limit: usize) -> Self {
            self.inner = self.inner.max_decoding_message_size(limit);
            self
        }
        /// Limits the maximum size of an encoded message.
        ///
        /// Default: `usize::MAX`
        #[must_use]
        pub fn max_encoding_message_size(mut self, limit: usize) -> Self {
            self.inner = self.inner.max_encoding_message_size(limit);
            self
        }
        pub async fn boolean(
            &mut self,
            request: impl tonic::IntoRequest<super::EvaluationRequest>,
        ) -> std::result::Result<
            tonic::Response<super::BooleanEvaluationResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/flipt.evaluation.EvaluationService/Boolean",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new("flipt.evaluation.EvaluationService", "Boolean"),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn variant(
            &mut self,
            request: impl tonic::IntoRequest<super::EvaluationRequest>,
        ) -> std::result::Result<
            tonic::Response<super::VariantEvaluationResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/flipt.evaluation.EvaluationService/Variant",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new("flipt.evaluation.EvaluationService", "Variant"),
                );
            self.inner.unary(req, path, codec).await
        }
        pub async fn batch(
            &mut self,
            request: impl tonic::IntoRequest<super::BatchEvaluationRequest>,
        ) -> std::result::Result<
            tonic::Response<super::BatchEvaluationResponse>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/flipt.evaluation.EvaluationService/Batch",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(GrpcMethod::new("flipt.evaluation.EvaluationService", "Batch"));
            self.inner.unary(req, path, codec).await
        }
    }
}
/// Generated client implementations.
pub mod data_service_client {
    #![allow(unused_variables, dead_code, missing_docs, clippy::let_unit_value)]
    use tonic::codegen::*;
    use tonic::codegen::http::Uri;
    /// flipt:sdk:ignore
    #[derive(Debug, Clone)]
    pub struct DataServiceClient<T> {
        inner: tonic::client::Grpc<T>,
    }
    impl DataServiceClient<tonic::transport::Channel> {
        /// Attempt to create a new client by connecting to a given endpoint.
        pub async fn connect<D>(dst: D) -> Result<Self, tonic::transport::Error>
        where
            D: TryInto<tonic::transport::Endpoint>,
            D::Error: Into<StdError>,
        {
            let conn = tonic::transport::Endpoint::new(dst)?.connect().await?;
            Ok(Self::new(conn))
        }
    }
    impl<T> DataServiceClient<T>
    where
        T: tonic::client::GrpcService<tonic::body::BoxBody>,
        T::Error: Into<StdError>,
        T::ResponseBody: Body<Data = Bytes> + Send + 'static,
        <T::ResponseBody as Body>::Error: Into<StdError> + Send,
    {
        pub fn new(inner: T) -> Self {
            let inner = tonic::client::Grpc::new(inner);
            Self { inner }
        }
        pub fn with_origin(inner: T, origin: Uri) -> Self {
            let inner = tonic::client::Grpc::with_origin(inner, origin);
            Self { inner }
        }
        pub fn with_interceptor<F>(
            inner: T,
            interceptor: F,
        ) -> DataServiceClient<InterceptedService<T, F>>
        where
            F: tonic::service::Interceptor,
            T::ResponseBody: Default,
            T: tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
                Response = http::Response<
                    <T as tonic::client::GrpcService<tonic::body::BoxBody>>::ResponseBody,
                >,
            >,
            <T as tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
            >>::Error: Into<StdError> + Send + Sync,
        {
            DataServiceClient::new(InterceptedService::new(inner, interceptor))
        }
        /// Compress requests with the given encoding.
        ///
        /// This requires the server to support it otherwise it might respond with an
        /// error.
        #[must_use]
        pub fn send_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.inner = self.inner.send_compressed(encoding);
            self
        }
        /// Enable decompressing responses.
        #[must_use]
        pub fn accept_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.inner = self.inner.accept_compressed(encoding);
            self
        }
        /// Limits the maximum size of a decoded message.
        ///
        /// Default: `4MB`
        #[must_use]
        pub fn max_decoding_message_size(mut self, limit: usize) -> Self {
            self.inner = self.inner.max_decoding_message_size(limit);
            self
        }
        /// Limits the maximum size of an encoded message.
        ///
        /// Default: `usize::MAX`
        #[must_use]
        pub fn max_encoding_message_size(mut self, limit: usize) -> Self {
            self.inner = self.inner.max_encoding_message_size(limit);
            self
        }
        pub async fn evaluation_snapshot_namespace(
            &mut self,
            request: impl tonic::IntoRequest<super::EvaluationNamespaceSnapshotRequest>,
        ) -> std::result::Result<
            tonic::Response<super::EvaluationNamespaceSnapshot>,
            tonic::Status,
        > {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/flipt.evaluation.DataService/EvaluationSnapshotNamespace",
            );
            let mut req = request.into_request();
            req.extensions_mut()
                .insert(
                    GrpcMethod::new(
                        "flipt.evaluation.DataService",
                        "EvaluationSnapshotNamespace",
                    ),
                );
            self.inner.unary(req, path, codec).await
        }
    }
}

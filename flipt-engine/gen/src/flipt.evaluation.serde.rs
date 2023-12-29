impl serde::Serialize for BatchEvaluationRequest {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.request_id.is_empty() {
            len += 1;
        }
        if !self.requests.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.BatchEvaluationRequest", len)?;
        if !self.request_id.is_empty() {
            struct_ser.serialize_field("requestId", &self.request_id)?;
        }
        if !self.requests.is_empty() {
            struct_ser.serialize_field("requests", &self.requests)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for BatchEvaluationRequest {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "request_id",
            "requestId",
            "requests",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            RequestId,
            Requests,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "requestId" | "request_id" => Ok(GeneratedField::RequestId),
                            "requests" => Ok(GeneratedField::Requests),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = BatchEvaluationRequest;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.BatchEvaluationRequest")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<BatchEvaluationRequest, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut request_id__ = None;
                let mut requests__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::RequestId => {
                            if request_id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("requestId"));
                            }
                            request_id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Requests => {
                            if requests__.is_some() {
                                return Err(serde::de::Error::duplicate_field("requests"));
                            }
                            requests__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(BatchEvaluationRequest {
                    request_id: request_id__.unwrap_or_default(),
                    requests: requests__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.BatchEvaluationRequest", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for BatchEvaluationResponse {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.request_id.is_empty() {
            len += 1;
        }
        if !self.responses.is_empty() {
            len += 1;
        }
        if self.request_duration_millis != 0. {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.BatchEvaluationResponse", len)?;
        if !self.request_id.is_empty() {
            struct_ser.serialize_field("requestId", &self.request_id)?;
        }
        if !self.responses.is_empty() {
            struct_ser.serialize_field("responses", &self.responses)?;
        }
        if self.request_duration_millis != 0. {
            struct_ser.serialize_field("requestDurationMillis", &self.request_duration_millis)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for BatchEvaluationResponse {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "request_id",
            "requestId",
            "responses",
            "request_duration_millis",
            "requestDurationMillis",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            RequestId,
            Responses,
            RequestDurationMillis,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "requestId" | "request_id" => Ok(GeneratedField::RequestId),
                            "responses" => Ok(GeneratedField::Responses),
                            "requestDurationMillis" | "request_duration_millis" => Ok(GeneratedField::RequestDurationMillis),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = BatchEvaluationResponse;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.BatchEvaluationResponse")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<BatchEvaluationResponse, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut request_id__ = None;
                let mut responses__ = None;
                let mut request_duration_millis__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::RequestId => {
                            if request_id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("requestId"));
                            }
                            request_id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Responses => {
                            if responses__.is_some() {
                                return Err(serde::de::Error::duplicate_field("responses"));
                            }
                            responses__ = Some(map_.next_value()?);
                        }
                        GeneratedField::RequestDurationMillis => {
                            if request_duration_millis__.is_some() {
                                return Err(serde::de::Error::duplicate_field("requestDurationMillis"));
                            }
                            request_duration_millis__ = 
                                Some(map_.next_value::<::pbjson::private::NumberDeserialize<_>>()?.0)
                            ;
                        }
                    }
                }
                Ok(BatchEvaluationResponse {
                    request_id: request_id__.unwrap_or_default(),
                    responses: responses__.unwrap_or_default(),
                    request_duration_millis: request_duration_millis__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.BatchEvaluationResponse", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for BooleanEvaluationResponse {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if self.enabled {
            len += 1;
        }
        if self.reason != 0 {
            len += 1;
        }
        if !self.request_id.is_empty() {
            len += 1;
        }
        if self.request_duration_millis != 0. {
            len += 1;
        }
        if self.timestamp.is_some() {
            len += 1;
        }
        if !self.flag_key.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.BooleanEvaluationResponse", len)?;
        if self.enabled {
            struct_ser.serialize_field("enabled", &self.enabled)?;
        }
        if self.reason != 0 {
            let v = EvaluationReason::try_from(self.reason)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.reason)))?;
            struct_ser.serialize_field("reason", &v)?;
        }
        if !self.request_id.is_empty() {
            struct_ser.serialize_field("requestId", &self.request_id)?;
        }
        if self.request_duration_millis != 0. {
            struct_ser.serialize_field("requestDurationMillis", &self.request_duration_millis)?;
        }
        if let Some(v) = self.timestamp.as_ref() {
            struct_ser.serialize_field("timestamp", v)?;
        }
        if !self.flag_key.is_empty() {
            struct_ser.serialize_field("flagKey", &self.flag_key)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for BooleanEvaluationResponse {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "enabled",
            "reason",
            "request_id",
            "requestId",
            "request_duration_millis",
            "requestDurationMillis",
            "timestamp",
            "flag_key",
            "flagKey",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Enabled,
            Reason,
            RequestId,
            RequestDurationMillis,
            Timestamp,
            FlagKey,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "enabled" => Ok(GeneratedField::Enabled),
                            "reason" => Ok(GeneratedField::Reason),
                            "requestId" | "request_id" => Ok(GeneratedField::RequestId),
                            "requestDurationMillis" | "request_duration_millis" => Ok(GeneratedField::RequestDurationMillis),
                            "timestamp" => Ok(GeneratedField::Timestamp),
                            "flagKey" | "flag_key" => Ok(GeneratedField::FlagKey),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = BooleanEvaluationResponse;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.BooleanEvaluationResponse")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<BooleanEvaluationResponse, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut enabled__ = None;
                let mut reason__ = None;
                let mut request_id__ = None;
                let mut request_duration_millis__ = None;
                let mut timestamp__ = None;
                let mut flag_key__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Enabled => {
                            if enabled__.is_some() {
                                return Err(serde::de::Error::duplicate_field("enabled"));
                            }
                            enabled__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Reason => {
                            if reason__.is_some() {
                                return Err(serde::de::Error::duplicate_field("reason"));
                            }
                            reason__ = Some(map_.next_value::<EvaluationReason>()? as i32);
                        }
                        GeneratedField::RequestId => {
                            if request_id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("requestId"));
                            }
                            request_id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::RequestDurationMillis => {
                            if request_duration_millis__.is_some() {
                                return Err(serde::de::Error::duplicate_field("requestDurationMillis"));
                            }
                            request_duration_millis__ = 
                                Some(map_.next_value::<::pbjson::private::NumberDeserialize<_>>()?.0)
                            ;
                        }
                        GeneratedField::Timestamp => {
                            if timestamp__.is_some() {
                                return Err(serde::de::Error::duplicate_field("timestamp"));
                            }
                            timestamp__ = map_.next_value()?;
                        }
                        GeneratedField::FlagKey => {
                            if flag_key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("flagKey"));
                            }
                            flag_key__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(BooleanEvaluationResponse {
                    enabled: enabled__.unwrap_or_default(),
                    reason: reason__.unwrap_or_default(),
                    request_id: request_id__.unwrap_or_default(),
                    request_duration_millis: request_duration_millis__.unwrap_or_default(),
                    timestamp: timestamp__,
                    flag_key: flag_key__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.BooleanEvaluationResponse", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for ErrorEvaluationReason {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let variant = match self {
            Self::UnknownErrorEvaluationReason => "UNKNOWN_ERROR_EVALUATION_REASON",
            Self::NotFoundErrorEvaluationReason => "NOT_FOUND_ERROR_EVALUATION_REASON",
        };
        serializer.serialize_str(variant)
    }
}
impl<'de> serde::Deserialize<'de> for ErrorEvaluationReason {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "UNKNOWN_ERROR_EVALUATION_REASON",
            "NOT_FOUND_ERROR_EVALUATION_REASON",
        ];

        struct GeneratedVisitor;

        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = ErrorEvaluationReason;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                write!(formatter, "expected one of: {:?}", &FIELDS)
            }

            fn visit_i64<E>(self, v: i64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Signed(v), &self)
                    })
            }

            fn visit_u64<E>(self, v: u64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Unsigned(v), &self)
                    })
            }

            fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                match value {
                    "UNKNOWN_ERROR_EVALUATION_REASON" => Ok(ErrorEvaluationReason::UnknownErrorEvaluationReason),
                    "NOT_FOUND_ERROR_EVALUATION_REASON" => Ok(ErrorEvaluationReason::NotFoundErrorEvaluationReason),
                    _ => Err(serde::de::Error::unknown_variant(value, FIELDS)),
                }
            }
        }
        deserializer.deserialize_any(GeneratedVisitor)
    }
}
impl serde::Serialize for ErrorEvaluationResponse {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.flag_key.is_empty() {
            len += 1;
        }
        if !self.namespace_key.is_empty() {
            len += 1;
        }
        if self.reason != 0 {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.ErrorEvaluationResponse", len)?;
        if !self.flag_key.is_empty() {
            struct_ser.serialize_field("flagKey", &self.flag_key)?;
        }
        if !self.namespace_key.is_empty() {
            struct_ser.serialize_field("namespaceKey", &self.namespace_key)?;
        }
        if self.reason != 0 {
            let v = ErrorEvaluationReason::try_from(self.reason)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.reason)))?;
            struct_ser.serialize_field("reason", &v)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for ErrorEvaluationResponse {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "flag_key",
            "flagKey",
            "namespace_key",
            "namespaceKey",
            "reason",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            FlagKey,
            NamespaceKey,
            Reason,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "flagKey" | "flag_key" => Ok(GeneratedField::FlagKey),
                            "namespaceKey" | "namespace_key" => Ok(GeneratedField::NamespaceKey),
                            "reason" => Ok(GeneratedField::Reason),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = ErrorEvaluationResponse;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.ErrorEvaluationResponse")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<ErrorEvaluationResponse, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut flag_key__ = None;
                let mut namespace_key__ = None;
                let mut reason__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::FlagKey => {
                            if flag_key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("flagKey"));
                            }
                            flag_key__ = Some(map_.next_value()?);
                        }
                        GeneratedField::NamespaceKey => {
                            if namespace_key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("namespaceKey"));
                            }
                            namespace_key__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Reason => {
                            if reason__.is_some() {
                                return Err(serde::de::Error::duplicate_field("reason"));
                            }
                            reason__ = Some(map_.next_value::<ErrorEvaluationReason>()? as i32);
                        }
                    }
                }
                Ok(ErrorEvaluationResponse {
                    flag_key: flag_key__.unwrap_or_default(),
                    namespace_key: namespace_key__.unwrap_or_default(),
                    reason: reason__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.ErrorEvaluationResponse", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationConstraint {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.id.is_empty() {
            len += 1;
        }
        if self.r#type != 0 {
            len += 1;
        }
        if !self.property.is_empty() {
            len += 1;
        }
        if !self.operator.is_empty() {
            len += 1;
        }
        if !self.value.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationConstraint", len)?;
        if !self.id.is_empty() {
            struct_ser.serialize_field("id", &self.id)?;
        }
        if self.r#type != 0 {
            let v = EvaluationConstraintComparisonType::try_from(self.r#type)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.r#type)))?;
            struct_ser.serialize_field("type", &v)?;
        }
        if !self.property.is_empty() {
            struct_ser.serialize_field("property", &self.property)?;
        }
        if !self.operator.is_empty() {
            struct_ser.serialize_field("operator", &self.operator)?;
        }
        if !self.value.is_empty() {
            struct_ser.serialize_field("value", &self.value)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationConstraint {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "id",
            "type",
            "property",
            "operator",
            "value",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Id,
            Type,
            Property,
            Operator,
            Value,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "id" => Ok(GeneratedField::Id),
                            "type" => Ok(GeneratedField::Type),
                            "property" => Ok(GeneratedField::Property),
                            "operator" => Ok(GeneratedField::Operator),
                            "value" => Ok(GeneratedField::Value),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationConstraint;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationConstraint")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationConstraint, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut id__ = None;
                let mut r#type__ = None;
                let mut property__ = None;
                let mut operator__ = None;
                let mut value__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Id => {
                            if id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("id"));
                            }
                            id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Type => {
                            if r#type__.is_some() {
                                return Err(serde::de::Error::duplicate_field("type"));
                            }
                            r#type__ = Some(map_.next_value::<EvaluationConstraintComparisonType>()? as i32);
                        }
                        GeneratedField::Property => {
                            if property__.is_some() {
                                return Err(serde::de::Error::duplicate_field("property"));
                            }
                            property__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Operator => {
                            if operator__.is_some() {
                                return Err(serde::de::Error::duplicate_field("operator"));
                            }
                            operator__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Value => {
                            if value__.is_some() {
                                return Err(serde::de::Error::duplicate_field("value"));
                            }
                            value__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(EvaluationConstraint {
                    id: id__.unwrap_or_default(),
                    r#type: r#type__.unwrap_or_default(),
                    property: property__.unwrap_or_default(),
                    operator: operator__.unwrap_or_default(),
                    value: value__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationConstraint", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationConstraintComparisonType {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let variant = match self {
            Self::UnknownConstraintComparisonType => "UNKNOWN_CONSTRAINT_COMPARISON_TYPE",
            Self::StringConstraintComparisonType => "STRING_CONSTRAINT_COMPARISON_TYPE",
            Self::NumberConstraintComparisonType => "NUMBER_CONSTRAINT_COMPARISON_TYPE",
            Self::BooleanConstraintComparisonType => "BOOLEAN_CONSTRAINT_COMPARISON_TYPE",
            Self::DatetimeConstraintComparisonType => "DATETIME_CONSTRAINT_COMPARISON_TYPE",
        };
        serializer.serialize_str(variant)
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationConstraintComparisonType {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "UNKNOWN_CONSTRAINT_COMPARISON_TYPE",
            "STRING_CONSTRAINT_COMPARISON_TYPE",
            "NUMBER_CONSTRAINT_COMPARISON_TYPE",
            "BOOLEAN_CONSTRAINT_COMPARISON_TYPE",
            "DATETIME_CONSTRAINT_COMPARISON_TYPE",
        ];

        struct GeneratedVisitor;

        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationConstraintComparisonType;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                write!(formatter, "expected one of: {:?}", &FIELDS)
            }

            fn visit_i64<E>(self, v: i64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Signed(v), &self)
                    })
            }

            fn visit_u64<E>(self, v: u64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Unsigned(v), &self)
                    })
            }

            fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                match value {
                    "UNKNOWN_CONSTRAINT_COMPARISON_TYPE" => Ok(EvaluationConstraintComparisonType::UnknownConstraintComparisonType),
                    "STRING_CONSTRAINT_COMPARISON_TYPE" => Ok(EvaluationConstraintComparisonType::StringConstraintComparisonType),
                    "NUMBER_CONSTRAINT_COMPARISON_TYPE" => Ok(EvaluationConstraintComparisonType::NumberConstraintComparisonType),
                    "BOOLEAN_CONSTRAINT_COMPARISON_TYPE" => Ok(EvaluationConstraintComparisonType::BooleanConstraintComparisonType),
                    "DATETIME_CONSTRAINT_COMPARISON_TYPE" => Ok(EvaluationConstraintComparisonType::DatetimeConstraintComparisonType),
                    _ => Err(serde::de::Error::unknown_variant(value, FIELDS)),
                }
            }
        }
        deserializer.deserialize_any(GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationDistribution {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.id.is_empty() {
            len += 1;
        }
        if !self.rule_id.is_empty() {
            len += 1;
        }
        if !self.variant_id.is_empty() {
            len += 1;
        }
        if !self.variant_key.is_empty() {
            len += 1;
        }
        if !self.variant_attachment.is_empty() {
            len += 1;
        }
        if self.rollout != 0. {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationDistribution", len)?;
        if !self.id.is_empty() {
            struct_ser.serialize_field("id", &self.id)?;
        }
        if !self.rule_id.is_empty() {
            struct_ser.serialize_field("ruleId", &self.rule_id)?;
        }
        if !self.variant_id.is_empty() {
            struct_ser.serialize_field("variantId", &self.variant_id)?;
        }
        if !self.variant_key.is_empty() {
            struct_ser.serialize_field("variantKey", &self.variant_key)?;
        }
        if !self.variant_attachment.is_empty() {
            struct_ser.serialize_field("variantAttachment", &self.variant_attachment)?;
        }
        if self.rollout != 0. {
            struct_ser.serialize_field("rollout", &self.rollout)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationDistribution {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "id",
            "rule_id",
            "ruleId",
            "variant_id",
            "variantId",
            "variant_key",
            "variantKey",
            "variant_attachment",
            "variantAttachment",
            "rollout",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Id,
            RuleId,
            VariantId,
            VariantKey,
            VariantAttachment,
            Rollout,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "id" => Ok(GeneratedField::Id),
                            "ruleId" | "rule_id" => Ok(GeneratedField::RuleId),
                            "variantId" | "variant_id" => Ok(GeneratedField::VariantId),
                            "variantKey" | "variant_key" => Ok(GeneratedField::VariantKey),
                            "variantAttachment" | "variant_attachment" => Ok(GeneratedField::VariantAttachment),
                            "rollout" => Ok(GeneratedField::Rollout),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationDistribution;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationDistribution")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationDistribution, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut id__ = None;
                let mut rule_id__ = None;
                let mut variant_id__ = None;
                let mut variant_key__ = None;
                let mut variant_attachment__ = None;
                let mut rollout__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Id => {
                            if id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("id"));
                            }
                            id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::RuleId => {
                            if rule_id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("ruleId"));
                            }
                            rule_id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::VariantId => {
                            if variant_id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("variantId"));
                            }
                            variant_id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::VariantKey => {
                            if variant_key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("variantKey"));
                            }
                            variant_key__ = Some(map_.next_value()?);
                        }
                        GeneratedField::VariantAttachment => {
                            if variant_attachment__.is_some() {
                                return Err(serde::de::Error::duplicate_field("variantAttachment"));
                            }
                            variant_attachment__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Rollout => {
                            if rollout__.is_some() {
                                return Err(serde::de::Error::duplicate_field("rollout"));
                            }
                            rollout__ = 
                                Some(map_.next_value::<::pbjson::private::NumberDeserialize<_>>()?.0)
                            ;
                        }
                    }
                }
                Ok(EvaluationDistribution {
                    id: id__.unwrap_or_default(),
                    rule_id: rule_id__.unwrap_or_default(),
                    variant_id: variant_id__.unwrap_or_default(),
                    variant_key: variant_key__.unwrap_or_default(),
                    variant_attachment: variant_attachment__.unwrap_or_default(),
                    rollout: rollout__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationDistribution", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationFlag {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.key.is_empty() {
            len += 1;
        }
        if !self.name.is_empty() {
            len += 1;
        }
        if !self.description.is_empty() {
            len += 1;
        }
        if self.enabled {
            len += 1;
        }
        if self.r#type != 0 {
            len += 1;
        }
        if self.created_at.is_some() {
            len += 1;
        }
        if self.updated_at.is_some() {
            len += 1;
        }
        if !self.rules.is_empty() {
            len += 1;
        }
        if !self.rollouts.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationFlag", len)?;
        if !self.key.is_empty() {
            struct_ser.serialize_field("key", &self.key)?;
        }
        if !self.name.is_empty() {
            struct_ser.serialize_field("name", &self.name)?;
        }
        if !self.description.is_empty() {
            struct_ser.serialize_field("description", &self.description)?;
        }
        if self.enabled {
            struct_ser.serialize_field("enabled", &self.enabled)?;
        }
        if self.r#type != 0 {
            let v = EvaluationFlagType::try_from(self.r#type)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.r#type)))?;
            struct_ser.serialize_field("type", &v)?;
        }
        if let Some(v) = self.created_at.as_ref() {
            struct_ser.serialize_field("createdAt", v)?;
        }
        if let Some(v) = self.updated_at.as_ref() {
            struct_ser.serialize_field("updatedAt", v)?;
        }
        if !self.rules.is_empty() {
            struct_ser.serialize_field("rules", &self.rules)?;
        }
        if !self.rollouts.is_empty() {
            struct_ser.serialize_field("rollouts", &self.rollouts)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationFlag {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "key",
            "name",
            "description",
            "enabled",
            "type",
            "created_at",
            "createdAt",
            "updated_at",
            "updatedAt",
            "rules",
            "rollouts",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Key,
            Name,
            Description,
            Enabled,
            Type,
            CreatedAt,
            UpdatedAt,
            Rules,
            Rollouts,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "key" => Ok(GeneratedField::Key),
                            "name" => Ok(GeneratedField::Name),
                            "description" => Ok(GeneratedField::Description),
                            "enabled" => Ok(GeneratedField::Enabled),
                            "type" => Ok(GeneratedField::Type),
                            "createdAt" | "created_at" => Ok(GeneratedField::CreatedAt),
                            "updatedAt" | "updated_at" => Ok(GeneratedField::UpdatedAt),
                            "rules" => Ok(GeneratedField::Rules),
                            "rollouts" => Ok(GeneratedField::Rollouts),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationFlag;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationFlag")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationFlag, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut key__ = None;
                let mut name__ = None;
                let mut description__ = None;
                let mut enabled__ = None;
                let mut r#type__ = None;
                let mut created_at__ = None;
                let mut updated_at__ = None;
                let mut rules__ = None;
                let mut rollouts__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Key => {
                            if key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("key"));
                            }
                            key__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Name => {
                            if name__.is_some() {
                                return Err(serde::de::Error::duplicate_field("name"));
                            }
                            name__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Description => {
                            if description__.is_some() {
                                return Err(serde::de::Error::duplicate_field("description"));
                            }
                            description__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Enabled => {
                            if enabled__.is_some() {
                                return Err(serde::de::Error::duplicate_field("enabled"));
                            }
                            enabled__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Type => {
                            if r#type__.is_some() {
                                return Err(serde::de::Error::duplicate_field("type"));
                            }
                            r#type__ = Some(map_.next_value::<EvaluationFlagType>()? as i32);
                        }
                        GeneratedField::CreatedAt => {
                            if created_at__.is_some() {
                                return Err(serde::de::Error::duplicate_field("createdAt"));
                            }
                            created_at__ = map_.next_value()?;
                        }
                        GeneratedField::UpdatedAt => {
                            if updated_at__.is_some() {
                                return Err(serde::de::Error::duplicate_field("updatedAt"));
                            }
                            updated_at__ = map_.next_value()?;
                        }
                        GeneratedField::Rules => {
                            if rules__.is_some() {
                                return Err(serde::de::Error::duplicate_field("rules"));
                            }
                            rules__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Rollouts => {
                            if rollouts__.is_some() {
                                return Err(serde::de::Error::duplicate_field("rollouts"));
                            }
                            rollouts__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(EvaluationFlag {
                    key: key__.unwrap_or_default(),
                    name: name__.unwrap_or_default(),
                    description: description__.unwrap_or_default(),
                    enabled: enabled__.unwrap_or_default(),
                    r#type: r#type__.unwrap_or_default(),
                    created_at: created_at__,
                    updated_at: updated_at__,
                    rules: rules__.unwrap_or_default(),
                    rollouts: rollouts__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationFlag", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationFlagType {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let variant = match self {
            Self::VariantFlagType => "VARIANT_FLAG_TYPE",
            Self::BooleanFlagType => "BOOLEAN_FLAG_TYPE",
        };
        serializer.serialize_str(variant)
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationFlagType {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "VARIANT_FLAG_TYPE",
            "BOOLEAN_FLAG_TYPE",
        ];

        struct GeneratedVisitor;

        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationFlagType;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                write!(formatter, "expected one of: {:?}", &FIELDS)
            }

            fn visit_i64<E>(self, v: i64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Signed(v), &self)
                    })
            }

            fn visit_u64<E>(self, v: u64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Unsigned(v), &self)
                    })
            }

            fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                match value {
                    "VARIANT_FLAG_TYPE" => Ok(EvaluationFlagType::VariantFlagType),
                    "BOOLEAN_FLAG_TYPE" => Ok(EvaluationFlagType::BooleanFlagType),
                    _ => Err(serde::de::Error::unknown_variant(value, FIELDS)),
                }
            }
        }
        deserializer.deserialize_any(GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationNamespace {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.key.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationNamespace", len)?;
        if !self.key.is_empty() {
            struct_ser.serialize_field("key", &self.key)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationNamespace {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "key",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Key,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "key" => Ok(GeneratedField::Key),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationNamespace;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationNamespace")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationNamespace, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut key__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Key => {
                            if key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("key"));
                            }
                            key__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(EvaluationNamespace {
                    key: key__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationNamespace", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationNamespaceSnapshot {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if self.namespace.is_some() {
            len += 1;
        }
        if !self.flags.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationNamespaceSnapshot", len)?;
        if let Some(v) = self.namespace.as_ref() {
            struct_ser.serialize_field("namespace", v)?;
        }
        if !self.flags.is_empty() {
            struct_ser.serialize_field("flags", &self.flags)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationNamespaceSnapshot {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "namespace",
            "flags",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Namespace,
            Flags,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "namespace" => Ok(GeneratedField::Namespace),
                            "flags" => Ok(GeneratedField::Flags),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationNamespaceSnapshot;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationNamespaceSnapshot")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationNamespaceSnapshot, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut namespace__ = None;
                let mut flags__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Namespace => {
                            if namespace__.is_some() {
                                return Err(serde::de::Error::duplicate_field("namespace"));
                            }
                            namespace__ = map_.next_value()?;
                        }
                        GeneratedField::Flags => {
                            if flags__.is_some() {
                                return Err(serde::de::Error::duplicate_field("flags"));
                            }
                            flags__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(EvaluationNamespaceSnapshot {
                    namespace: namespace__,
                    flags: flags__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationNamespaceSnapshot", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationNamespaceSnapshotRequest {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.key.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationNamespaceSnapshotRequest", len)?;
        if !self.key.is_empty() {
            struct_ser.serialize_field("key", &self.key)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationNamespaceSnapshotRequest {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "key",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Key,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "key" => Ok(GeneratedField::Key),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationNamespaceSnapshotRequest;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationNamespaceSnapshotRequest")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationNamespaceSnapshotRequest, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut key__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Key => {
                            if key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("key"));
                            }
                            key__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(EvaluationNamespaceSnapshotRequest {
                    key: key__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationNamespaceSnapshotRequest", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationReason {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let variant = match self {
            Self::UnknownEvaluationReason => "UNKNOWN_EVALUATION_REASON",
            Self::FlagDisabledEvaluationReason => "FLAG_DISABLED_EVALUATION_REASON",
            Self::MatchEvaluationReason => "MATCH_EVALUATION_REASON",
            Self::DefaultEvaluationReason => "DEFAULT_EVALUATION_REASON",
        };
        serializer.serialize_str(variant)
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationReason {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "UNKNOWN_EVALUATION_REASON",
            "FLAG_DISABLED_EVALUATION_REASON",
            "MATCH_EVALUATION_REASON",
            "DEFAULT_EVALUATION_REASON",
        ];

        struct GeneratedVisitor;

        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationReason;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                write!(formatter, "expected one of: {:?}", &FIELDS)
            }

            fn visit_i64<E>(self, v: i64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Signed(v), &self)
                    })
            }

            fn visit_u64<E>(self, v: u64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Unsigned(v), &self)
                    })
            }

            fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                match value {
                    "UNKNOWN_EVALUATION_REASON" => Ok(EvaluationReason::UnknownEvaluationReason),
                    "FLAG_DISABLED_EVALUATION_REASON" => Ok(EvaluationReason::FlagDisabledEvaluationReason),
                    "MATCH_EVALUATION_REASON" => Ok(EvaluationReason::MatchEvaluationReason),
                    "DEFAULT_EVALUATION_REASON" => Ok(EvaluationReason::DefaultEvaluationReason),
                    _ => Err(serde::de::Error::unknown_variant(value, FIELDS)),
                }
            }
        }
        deserializer.deserialize_any(GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationRequest {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.request_id.is_empty() {
            len += 1;
        }
        if !self.namespace_key.is_empty() {
            len += 1;
        }
        if !self.flag_key.is_empty() {
            len += 1;
        }
        if !self.entity_id.is_empty() {
            len += 1;
        }
        if !self.context.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationRequest", len)?;
        if !self.request_id.is_empty() {
            struct_ser.serialize_field("requestId", &self.request_id)?;
        }
        if !self.namespace_key.is_empty() {
            struct_ser.serialize_field("namespaceKey", &self.namespace_key)?;
        }
        if !self.flag_key.is_empty() {
            struct_ser.serialize_field("flagKey", &self.flag_key)?;
        }
        if !self.entity_id.is_empty() {
            struct_ser.serialize_field("entityId", &self.entity_id)?;
        }
        if !self.context.is_empty() {
            struct_ser.serialize_field("context", &self.context)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationRequest {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "request_id",
            "requestId",
            "namespace_key",
            "namespaceKey",
            "flag_key",
            "flagKey",
            "entity_id",
            "entityId",
            "context",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            RequestId,
            NamespaceKey,
            FlagKey,
            EntityId,
            Context,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "requestId" | "request_id" => Ok(GeneratedField::RequestId),
                            "namespaceKey" | "namespace_key" => Ok(GeneratedField::NamespaceKey),
                            "flagKey" | "flag_key" => Ok(GeneratedField::FlagKey),
                            "entityId" | "entity_id" => Ok(GeneratedField::EntityId),
                            "context" => Ok(GeneratedField::Context),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationRequest;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationRequest")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationRequest, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut request_id__ = None;
                let mut namespace_key__ = None;
                let mut flag_key__ = None;
                let mut entity_id__ = None;
                let mut context__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::RequestId => {
                            if request_id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("requestId"));
                            }
                            request_id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::NamespaceKey => {
                            if namespace_key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("namespaceKey"));
                            }
                            namespace_key__ = Some(map_.next_value()?);
                        }
                        GeneratedField::FlagKey => {
                            if flag_key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("flagKey"));
                            }
                            flag_key__ = Some(map_.next_value()?);
                        }
                        GeneratedField::EntityId => {
                            if entity_id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("entityId"));
                            }
                            entity_id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Context => {
                            if context__.is_some() {
                                return Err(serde::de::Error::duplicate_field("context"));
                            }
                            context__ = Some(
                                map_.next_value::<std::collections::HashMap<_, _>>()?
                            );
                        }
                    }
                }
                Ok(EvaluationRequest {
                    request_id: request_id__.unwrap_or_default(),
                    namespace_key: namespace_key__.unwrap_or_default(),
                    flag_key: flag_key__.unwrap_or_default(),
                    entity_id: entity_id__.unwrap_or_default(),
                    context: context__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationRequest", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationResponse {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if self.r#type != 0 {
            len += 1;
        }
        if self.response.is_some() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationResponse", len)?;
        if self.r#type != 0 {
            let v = EvaluationResponseType::try_from(self.r#type)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.r#type)))?;
            struct_ser.serialize_field("type", &v)?;
        }
        if let Some(v) = self.response.as_ref() {
            match v {
                evaluation_response::Response::BooleanResponse(v) => {
                    struct_ser.serialize_field("booleanResponse", v)?;
                }
                evaluation_response::Response::VariantResponse(v) => {
                    struct_ser.serialize_field("variantResponse", v)?;
                }
                evaluation_response::Response::ErrorResponse(v) => {
                    struct_ser.serialize_field("errorResponse", v)?;
                }
            }
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationResponse {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "type",
            "boolean_response",
            "booleanResponse",
            "variant_response",
            "variantResponse",
            "error_response",
            "errorResponse",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Type,
            BooleanResponse,
            VariantResponse,
            ErrorResponse,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "type" => Ok(GeneratedField::Type),
                            "booleanResponse" | "boolean_response" => Ok(GeneratedField::BooleanResponse),
                            "variantResponse" | "variant_response" => Ok(GeneratedField::VariantResponse),
                            "errorResponse" | "error_response" => Ok(GeneratedField::ErrorResponse),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationResponse;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationResponse")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationResponse, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut r#type__ = None;
                let mut response__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Type => {
                            if r#type__.is_some() {
                                return Err(serde::de::Error::duplicate_field("type"));
                            }
                            r#type__ = Some(map_.next_value::<EvaluationResponseType>()? as i32);
                        }
                        GeneratedField::BooleanResponse => {
                            if response__.is_some() {
                                return Err(serde::de::Error::duplicate_field("booleanResponse"));
                            }
                            response__ = map_.next_value::<::std::option::Option<_>>()?.map(evaluation_response::Response::BooleanResponse)
;
                        }
                        GeneratedField::VariantResponse => {
                            if response__.is_some() {
                                return Err(serde::de::Error::duplicate_field("variantResponse"));
                            }
                            response__ = map_.next_value::<::std::option::Option<_>>()?.map(evaluation_response::Response::VariantResponse)
;
                        }
                        GeneratedField::ErrorResponse => {
                            if response__.is_some() {
                                return Err(serde::de::Error::duplicate_field("errorResponse"));
                            }
                            response__ = map_.next_value::<::std::option::Option<_>>()?.map(evaluation_response::Response::ErrorResponse)
;
                        }
                    }
                }
                Ok(EvaluationResponse {
                    r#type: r#type__.unwrap_or_default(),
                    response: response__,
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationResponse", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationResponseType {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let variant = match self {
            Self::VariantEvaluationResponseType => "VARIANT_EVALUATION_RESPONSE_TYPE",
            Self::BooleanEvaluationResponseType => "BOOLEAN_EVALUATION_RESPONSE_TYPE",
            Self::ErrorEvaluationResponseType => "ERROR_EVALUATION_RESPONSE_TYPE",
        };
        serializer.serialize_str(variant)
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationResponseType {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "VARIANT_EVALUATION_RESPONSE_TYPE",
            "BOOLEAN_EVALUATION_RESPONSE_TYPE",
            "ERROR_EVALUATION_RESPONSE_TYPE",
        ];

        struct GeneratedVisitor;

        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationResponseType;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                write!(formatter, "expected one of: {:?}", &FIELDS)
            }

            fn visit_i64<E>(self, v: i64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Signed(v), &self)
                    })
            }

            fn visit_u64<E>(self, v: u64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Unsigned(v), &self)
                    })
            }

            fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                match value {
                    "VARIANT_EVALUATION_RESPONSE_TYPE" => Ok(EvaluationResponseType::VariantEvaluationResponseType),
                    "BOOLEAN_EVALUATION_RESPONSE_TYPE" => Ok(EvaluationResponseType::BooleanEvaluationResponseType),
                    "ERROR_EVALUATION_RESPONSE_TYPE" => Ok(EvaluationResponseType::ErrorEvaluationResponseType),
                    _ => Err(serde::de::Error::unknown_variant(value, FIELDS)),
                }
            }
        }
        deserializer.deserialize_any(GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationRollout {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if self.r#type != 0 {
            len += 1;
        }
        if self.rank != 0 {
            len += 1;
        }
        if self.rule.is_some() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationRollout", len)?;
        if self.r#type != 0 {
            let v = EvaluationRolloutType::try_from(self.r#type)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.r#type)))?;
            struct_ser.serialize_field("type", &v)?;
        }
        if self.rank != 0 {
            struct_ser.serialize_field("rank", &self.rank)?;
        }
        if let Some(v) = self.rule.as_ref() {
            match v {
                evaluation_rollout::Rule::Segment(v) => {
                    struct_ser.serialize_field("segment", v)?;
                }
                evaluation_rollout::Rule::Threshold(v) => {
                    struct_ser.serialize_field("threshold", v)?;
                }
            }
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationRollout {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "type",
            "rank",
            "segment",
            "threshold",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Type,
            Rank,
            Segment,
            Threshold,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "type" => Ok(GeneratedField::Type),
                            "rank" => Ok(GeneratedField::Rank),
                            "segment" => Ok(GeneratedField::Segment),
                            "threshold" => Ok(GeneratedField::Threshold),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationRollout;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationRollout")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationRollout, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut r#type__ = None;
                let mut rank__ = None;
                let mut rule__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Type => {
                            if r#type__.is_some() {
                                return Err(serde::de::Error::duplicate_field("type"));
                            }
                            r#type__ = Some(map_.next_value::<EvaluationRolloutType>()? as i32);
                        }
                        GeneratedField::Rank => {
                            if rank__.is_some() {
                                return Err(serde::de::Error::duplicate_field("rank"));
                            }
                            rank__ = 
                                Some(map_.next_value::<::pbjson::private::NumberDeserialize<_>>()?.0)
                            ;
                        }
                        GeneratedField::Segment => {
                            if rule__.is_some() {
                                return Err(serde::de::Error::duplicate_field("segment"));
                            }
                            rule__ = map_.next_value::<::std::option::Option<_>>()?.map(evaluation_rollout::Rule::Segment)
;
                        }
                        GeneratedField::Threshold => {
                            if rule__.is_some() {
                                return Err(serde::de::Error::duplicate_field("threshold"));
                            }
                            rule__ = map_.next_value::<::std::option::Option<_>>()?.map(evaluation_rollout::Rule::Threshold)
;
                        }
                    }
                }
                Ok(EvaluationRollout {
                    r#type: r#type__.unwrap_or_default(),
                    rank: rank__.unwrap_or_default(),
                    rule: rule__,
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationRollout", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationRolloutSegment {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if self.value {
            len += 1;
        }
        if self.segment_operator != 0 {
            len += 1;
        }
        if !self.segments.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationRolloutSegment", len)?;
        if self.value {
            struct_ser.serialize_field("value", &self.value)?;
        }
        if self.segment_operator != 0 {
            let v = EvaluationSegmentOperator::try_from(self.segment_operator)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.segment_operator)))?;
            struct_ser.serialize_field("segmentOperator", &v)?;
        }
        if !self.segments.is_empty() {
            struct_ser.serialize_field("segments", &self.segments)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationRolloutSegment {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "value",
            "segment_operator",
            "segmentOperator",
            "segments",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Value,
            SegmentOperator,
            Segments,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "value" => Ok(GeneratedField::Value),
                            "segmentOperator" | "segment_operator" => Ok(GeneratedField::SegmentOperator),
                            "segments" => Ok(GeneratedField::Segments),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationRolloutSegment;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationRolloutSegment")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationRolloutSegment, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut value__ = None;
                let mut segment_operator__ = None;
                let mut segments__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Value => {
                            if value__.is_some() {
                                return Err(serde::de::Error::duplicate_field("value"));
                            }
                            value__ = Some(map_.next_value()?);
                        }
                        GeneratedField::SegmentOperator => {
                            if segment_operator__.is_some() {
                                return Err(serde::de::Error::duplicate_field("segmentOperator"));
                            }
                            segment_operator__ = Some(map_.next_value::<EvaluationSegmentOperator>()? as i32);
                        }
                        GeneratedField::Segments => {
                            if segments__.is_some() {
                                return Err(serde::de::Error::duplicate_field("segments"));
                            }
                            segments__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(EvaluationRolloutSegment {
                    value: value__.unwrap_or_default(),
                    segment_operator: segment_operator__.unwrap_or_default(),
                    segments: segments__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationRolloutSegment", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationRolloutThreshold {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if self.percentage != 0. {
            len += 1;
        }
        if self.value {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationRolloutThreshold", len)?;
        if self.percentage != 0. {
            struct_ser.serialize_field("percentage", &self.percentage)?;
        }
        if self.value {
            struct_ser.serialize_field("value", &self.value)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationRolloutThreshold {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "percentage",
            "value",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Percentage,
            Value,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "percentage" => Ok(GeneratedField::Percentage),
                            "value" => Ok(GeneratedField::Value),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationRolloutThreshold;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationRolloutThreshold")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationRolloutThreshold, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut percentage__ = None;
                let mut value__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Percentage => {
                            if percentage__.is_some() {
                                return Err(serde::de::Error::duplicate_field("percentage"));
                            }
                            percentage__ = 
                                Some(map_.next_value::<::pbjson::private::NumberDeserialize<_>>()?.0)
                            ;
                        }
                        GeneratedField::Value => {
                            if value__.is_some() {
                                return Err(serde::de::Error::duplicate_field("value"));
                            }
                            value__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(EvaluationRolloutThreshold {
                    percentage: percentage__.unwrap_or_default(),
                    value: value__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationRolloutThreshold", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationRolloutType {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let variant = match self {
            Self::UnknownRolloutType => "UNKNOWN_ROLLOUT_TYPE",
            Self::SegmentRolloutType => "SEGMENT_ROLLOUT_TYPE",
            Self::ThresholdRolloutType => "THRESHOLD_ROLLOUT_TYPE",
        };
        serializer.serialize_str(variant)
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationRolloutType {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "UNKNOWN_ROLLOUT_TYPE",
            "SEGMENT_ROLLOUT_TYPE",
            "THRESHOLD_ROLLOUT_TYPE",
        ];

        struct GeneratedVisitor;

        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationRolloutType;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                write!(formatter, "expected one of: {:?}", &FIELDS)
            }

            fn visit_i64<E>(self, v: i64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Signed(v), &self)
                    })
            }

            fn visit_u64<E>(self, v: u64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Unsigned(v), &self)
                    })
            }

            fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                match value {
                    "UNKNOWN_ROLLOUT_TYPE" => Ok(EvaluationRolloutType::UnknownRolloutType),
                    "SEGMENT_ROLLOUT_TYPE" => Ok(EvaluationRolloutType::SegmentRolloutType),
                    "THRESHOLD_ROLLOUT_TYPE" => Ok(EvaluationRolloutType::ThresholdRolloutType),
                    _ => Err(serde::de::Error::unknown_variant(value, FIELDS)),
                }
            }
        }
        deserializer.deserialize_any(GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationRule {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.id.is_empty() {
            len += 1;
        }
        if !self.segments.is_empty() {
            len += 1;
        }
        if self.rank != 0 {
            len += 1;
        }
        if self.segment_operator != 0 {
            len += 1;
        }
        if !self.distributions.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationRule", len)?;
        if !self.id.is_empty() {
            struct_ser.serialize_field("id", &self.id)?;
        }
        if !self.segments.is_empty() {
            struct_ser.serialize_field("segments", &self.segments)?;
        }
        if self.rank != 0 {
            struct_ser.serialize_field("rank", &self.rank)?;
        }
        if self.segment_operator != 0 {
            let v = EvaluationSegmentOperator::try_from(self.segment_operator)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.segment_operator)))?;
            struct_ser.serialize_field("segmentOperator", &v)?;
        }
        if !self.distributions.is_empty() {
            struct_ser.serialize_field("distributions", &self.distributions)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationRule {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "id",
            "segments",
            "rank",
            "segment_operator",
            "segmentOperator",
            "distributions",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Id,
            Segments,
            Rank,
            SegmentOperator,
            Distributions,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "id" => Ok(GeneratedField::Id),
                            "segments" => Ok(GeneratedField::Segments),
                            "rank" => Ok(GeneratedField::Rank),
                            "segmentOperator" | "segment_operator" => Ok(GeneratedField::SegmentOperator),
                            "distributions" => Ok(GeneratedField::Distributions),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationRule;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationRule")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationRule, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut id__ = None;
                let mut segments__ = None;
                let mut rank__ = None;
                let mut segment_operator__ = None;
                let mut distributions__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Id => {
                            if id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("id"));
                            }
                            id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Segments => {
                            if segments__.is_some() {
                                return Err(serde::de::Error::duplicate_field("segments"));
                            }
                            segments__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Rank => {
                            if rank__.is_some() {
                                return Err(serde::de::Error::duplicate_field("rank"));
                            }
                            rank__ = 
                                Some(map_.next_value::<::pbjson::private::NumberDeserialize<_>>()?.0)
                            ;
                        }
                        GeneratedField::SegmentOperator => {
                            if segment_operator__.is_some() {
                                return Err(serde::de::Error::duplicate_field("segmentOperator"));
                            }
                            segment_operator__ = Some(map_.next_value::<EvaluationSegmentOperator>()? as i32);
                        }
                        GeneratedField::Distributions => {
                            if distributions__.is_some() {
                                return Err(serde::de::Error::duplicate_field("distributions"));
                            }
                            distributions__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(EvaluationRule {
                    id: id__.unwrap_or_default(),
                    segments: segments__.unwrap_or_default(),
                    rank: rank__.unwrap_or_default(),
                    segment_operator: segment_operator__.unwrap_or_default(),
                    distributions: distributions__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationRule", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationSegment {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if !self.key.is_empty() {
            len += 1;
        }
        if !self.name.is_empty() {
            len += 1;
        }
        if !self.description.is_empty() {
            len += 1;
        }
        if self.match_type != 0 {
            len += 1;
        }
        if self.created_at.is_some() {
            len += 1;
        }
        if self.updated_at.is_some() {
            len += 1;
        }
        if !self.constraints.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.EvaluationSegment", len)?;
        if !self.key.is_empty() {
            struct_ser.serialize_field("key", &self.key)?;
        }
        if !self.name.is_empty() {
            struct_ser.serialize_field("name", &self.name)?;
        }
        if !self.description.is_empty() {
            struct_ser.serialize_field("description", &self.description)?;
        }
        if self.match_type != 0 {
            let v = EvaluationSegmentMatchType::try_from(self.match_type)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.match_type)))?;
            struct_ser.serialize_field("matchType", &v)?;
        }
        if let Some(v) = self.created_at.as_ref() {
            struct_ser.serialize_field("createdAt", v)?;
        }
        if let Some(v) = self.updated_at.as_ref() {
            struct_ser.serialize_field("updatedAt", v)?;
        }
        if !self.constraints.is_empty() {
            struct_ser.serialize_field("constraints", &self.constraints)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationSegment {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "key",
            "name",
            "description",
            "match_type",
            "matchType",
            "created_at",
            "createdAt",
            "updated_at",
            "updatedAt",
            "constraints",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Key,
            Name,
            Description,
            MatchType,
            CreatedAt,
            UpdatedAt,
            Constraints,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "key" => Ok(GeneratedField::Key),
                            "name" => Ok(GeneratedField::Name),
                            "description" => Ok(GeneratedField::Description),
                            "matchType" | "match_type" => Ok(GeneratedField::MatchType),
                            "createdAt" | "created_at" => Ok(GeneratedField::CreatedAt),
                            "updatedAt" | "updated_at" => Ok(GeneratedField::UpdatedAt),
                            "constraints" => Ok(GeneratedField::Constraints),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationSegment;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.EvaluationSegment")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<EvaluationSegment, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut key__ = None;
                let mut name__ = None;
                let mut description__ = None;
                let mut match_type__ = None;
                let mut created_at__ = None;
                let mut updated_at__ = None;
                let mut constraints__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Key => {
                            if key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("key"));
                            }
                            key__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Name => {
                            if name__.is_some() {
                                return Err(serde::de::Error::duplicate_field("name"));
                            }
                            name__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Description => {
                            if description__.is_some() {
                                return Err(serde::de::Error::duplicate_field("description"));
                            }
                            description__ = Some(map_.next_value()?);
                        }
                        GeneratedField::MatchType => {
                            if match_type__.is_some() {
                                return Err(serde::de::Error::duplicate_field("matchType"));
                            }
                            match_type__ = Some(map_.next_value::<EvaluationSegmentMatchType>()? as i32);
                        }
                        GeneratedField::CreatedAt => {
                            if created_at__.is_some() {
                                return Err(serde::de::Error::duplicate_field("createdAt"));
                            }
                            created_at__ = map_.next_value()?;
                        }
                        GeneratedField::UpdatedAt => {
                            if updated_at__.is_some() {
                                return Err(serde::de::Error::duplicate_field("updatedAt"));
                            }
                            updated_at__ = map_.next_value()?;
                        }
                        GeneratedField::Constraints => {
                            if constraints__.is_some() {
                                return Err(serde::de::Error::duplicate_field("constraints"));
                            }
                            constraints__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(EvaluationSegment {
                    key: key__.unwrap_or_default(),
                    name: name__.unwrap_or_default(),
                    description: description__.unwrap_or_default(),
                    match_type: match_type__.unwrap_or_default(),
                    created_at: created_at__,
                    updated_at: updated_at__,
                    constraints: constraints__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.EvaluationSegment", FIELDS, GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationSegmentMatchType {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let variant = match self {
            Self::AllSegmentMatchType => "ALL_SEGMENT_MATCH_TYPE",
            Self::AnySegmentMatchType => "ANY_SEGMENT_MATCH_TYPE",
        };
        serializer.serialize_str(variant)
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationSegmentMatchType {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "ALL_SEGMENT_MATCH_TYPE",
            "ANY_SEGMENT_MATCH_TYPE",
        ];

        struct GeneratedVisitor;

        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationSegmentMatchType;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                write!(formatter, "expected one of: {:?}", &FIELDS)
            }

            fn visit_i64<E>(self, v: i64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Signed(v), &self)
                    })
            }

            fn visit_u64<E>(self, v: u64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Unsigned(v), &self)
                    })
            }

            fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                match value {
                    "ALL_SEGMENT_MATCH_TYPE" => Ok(EvaluationSegmentMatchType::AllSegmentMatchType),
                    "ANY_SEGMENT_MATCH_TYPE" => Ok(EvaluationSegmentMatchType::AnySegmentMatchType),
                    _ => Err(serde::de::Error::unknown_variant(value, FIELDS)),
                }
            }
        }
        deserializer.deserialize_any(GeneratedVisitor)
    }
}
impl serde::Serialize for EvaluationSegmentOperator {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        let variant = match self {
            Self::OrSegmentOperator => "OR_SEGMENT_OPERATOR",
            Self::AndSegmentOperator => "AND_SEGMENT_OPERATOR",
        };
        serializer.serialize_str(variant)
    }
}
impl<'de> serde::Deserialize<'de> for EvaluationSegmentOperator {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "OR_SEGMENT_OPERATOR",
            "AND_SEGMENT_OPERATOR",
        ];

        struct GeneratedVisitor;

        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = EvaluationSegmentOperator;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                write!(formatter, "expected one of: {:?}", &FIELDS)
            }

            fn visit_i64<E>(self, v: i64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Signed(v), &self)
                    })
            }

            fn visit_u64<E>(self, v: u64) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                i32::try_from(v)
                    .ok()
                    .and_then(|x| x.try_into().ok())
                    .ok_or_else(|| {
                        serde::de::Error::invalid_value(serde::de::Unexpected::Unsigned(v), &self)
                    })
            }

            fn visit_str<E>(self, value: &str) -> std::result::Result<Self::Value, E>
            where
                E: serde::de::Error,
            {
                match value {
                    "OR_SEGMENT_OPERATOR" => Ok(EvaluationSegmentOperator::OrSegmentOperator),
                    "AND_SEGMENT_OPERATOR" => Ok(EvaluationSegmentOperator::AndSegmentOperator),
                    _ => Err(serde::de::Error::unknown_variant(value, FIELDS)),
                }
            }
        }
        deserializer.deserialize_any(GeneratedVisitor)
    }
}
impl serde::Serialize for VariantEvaluationResponse {
    #[allow(deprecated)]
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut len = 0;
        if self.r#match {
            len += 1;
        }
        if !self.segment_keys.is_empty() {
            len += 1;
        }
        if self.reason != 0 {
            len += 1;
        }
        if !self.variant_key.is_empty() {
            len += 1;
        }
        if !self.variant_attachment.is_empty() {
            len += 1;
        }
        if !self.request_id.is_empty() {
            len += 1;
        }
        if self.request_duration_millis != 0. {
            len += 1;
        }
        if self.timestamp.is_some() {
            len += 1;
        }
        if !self.flag_key.is_empty() {
            len += 1;
        }
        let mut struct_ser = serializer.serialize_struct("flipt.evaluation.VariantEvaluationResponse", len)?;
        if self.r#match {
            struct_ser.serialize_field("match", &self.r#match)?;
        }
        if !self.segment_keys.is_empty() {
            struct_ser.serialize_field("segmentKeys", &self.segment_keys)?;
        }
        if self.reason != 0 {
            let v = EvaluationReason::try_from(self.reason)
                .map_err(|_| serde::ser::Error::custom(format!("Invalid variant {}", self.reason)))?;
            struct_ser.serialize_field("reason", &v)?;
        }
        if !self.variant_key.is_empty() {
            struct_ser.serialize_field("variantKey", &self.variant_key)?;
        }
        if !self.variant_attachment.is_empty() {
            struct_ser.serialize_field("variantAttachment", &self.variant_attachment)?;
        }
        if !self.request_id.is_empty() {
            struct_ser.serialize_field("requestId", &self.request_id)?;
        }
        if self.request_duration_millis != 0. {
            struct_ser.serialize_field("requestDurationMillis", &self.request_duration_millis)?;
        }
        if let Some(v) = self.timestamp.as_ref() {
            struct_ser.serialize_field("timestamp", v)?;
        }
        if !self.flag_key.is_empty() {
            struct_ser.serialize_field("flagKey", &self.flag_key)?;
        }
        struct_ser.end()
    }
}
impl<'de> serde::Deserialize<'de> for VariantEvaluationResponse {
    #[allow(deprecated)]
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        const FIELDS: &[&str] = &[
            "match",
            "segment_keys",
            "segmentKeys",
            "reason",
            "variant_key",
            "variantKey",
            "variant_attachment",
            "variantAttachment",
            "request_id",
            "requestId",
            "request_duration_millis",
            "requestDurationMillis",
            "timestamp",
            "flag_key",
            "flagKey",
        ];

        #[allow(clippy::enum_variant_names)]
        enum GeneratedField {
            Match,
            SegmentKeys,
            Reason,
            VariantKey,
            VariantAttachment,
            RequestId,
            RequestDurationMillis,
            Timestamp,
            FlagKey,
        }
        impl<'de> serde::Deserialize<'de> for GeneratedField {
            fn deserialize<D>(deserializer: D) -> std::result::Result<GeneratedField, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                struct GeneratedVisitor;

                impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
                    type Value = GeneratedField;

                    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                        write!(formatter, "expected one of: {:?}", &FIELDS)
                    }

                    #[allow(unused_variables)]
                    fn visit_str<E>(self, value: &str) -> std::result::Result<GeneratedField, E>
                    where
                        E: serde::de::Error,
                    {
                        match value {
                            "match" => Ok(GeneratedField::Match),
                            "segmentKeys" | "segment_keys" => Ok(GeneratedField::SegmentKeys),
                            "reason" => Ok(GeneratedField::Reason),
                            "variantKey" | "variant_key" => Ok(GeneratedField::VariantKey),
                            "variantAttachment" | "variant_attachment" => Ok(GeneratedField::VariantAttachment),
                            "requestId" | "request_id" => Ok(GeneratedField::RequestId),
                            "requestDurationMillis" | "request_duration_millis" => Ok(GeneratedField::RequestDurationMillis),
                            "timestamp" => Ok(GeneratedField::Timestamp),
                            "flagKey" | "flag_key" => Ok(GeneratedField::FlagKey),
                            _ => Err(serde::de::Error::unknown_field(value, FIELDS)),
                        }
                    }
                }
                deserializer.deserialize_identifier(GeneratedVisitor)
            }
        }
        struct GeneratedVisitor;
        impl<'de> serde::de::Visitor<'de> for GeneratedVisitor {
            type Value = VariantEvaluationResponse;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("struct flipt.evaluation.VariantEvaluationResponse")
            }

            fn visit_map<V>(self, mut map_: V) -> std::result::Result<VariantEvaluationResponse, V::Error>
                where
                    V: serde::de::MapAccess<'de>,
            {
                let mut r#match__ = None;
                let mut segment_keys__ = None;
                let mut reason__ = None;
                let mut variant_key__ = None;
                let mut variant_attachment__ = None;
                let mut request_id__ = None;
                let mut request_duration_millis__ = None;
                let mut timestamp__ = None;
                let mut flag_key__ = None;
                while let Some(k) = map_.next_key()? {
                    match k {
                        GeneratedField::Match => {
                            if r#match__.is_some() {
                                return Err(serde::de::Error::duplicate_field("match"));
                            }
                            r#match__ = Some(map_.next_value()?);
                        }
                        GeneratedField::SegmentKeys => {
                            if segment_keys__.is_some() {
                                return Err(serde::de::Error::duplicate_field("segmentKeys"));
                            }
                            segment_keys__ = Some(map_.next_value()?);
                        }
                        GeneratedField::Reason => {
                            if reason__.is_some() {
                                return Err(serde::de::Error::duplicate_field("reason"));
                            }
                            reason__ = Some(map_.next_value::<EvaluationReason>()? as i32);
                        }
                        GeneratedField::VariantKey => {
                            if variant_key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("variantKey"));
                            }
                            variant_key__ = Some(map_.next_value()?);
                        }
                        GeneratedField::VariantAttachment => {
                            if variant_attachment__.is_some() {
                                return Err(serde::de::Error::duplicate_field("variantAttachment"));
                            }
                            variant_attachment__ = Some(map_.next_value()?);
                        }
                        GeneratedField::RequestId => {
                            if request_id__.is_some() {
                                return Err(serde::de::Error::duplicate_field("requestId"));
                            }
                            request_id__ = Some(map_.next_value()?);
                        }
                        GeneratedField::RequestDurationMillis => {
                            if request_duration_millis__.is_some() {
                                return Err(serde::de::Error::duplicate_field("requestDurationMillis"));
                            }
                            request_duration_millis__ = 
                                Some(map_.next_value::<::pbjson::private::NumberDeserialize<_>>()?.0)
                            ;
                        }
                        GeneratedField::Timestamp => {
                            if timestamp__.is_some() {
                                return Err(serde::de::Error::duplicate_field("timestamp"));
                            }
                            timestamp__ = map_.next_value()?;
                        }
                        GeneratedField::FlagKey => {
                            if flag_key__.is_some() {
                                return Err(serde::de::Error::duplicate_field("flagKey"));
                            }
                            flag_key__ = Some(map_.next_value()?);
                        }
                    }
                }
                Ok(VariantEvaluationResponse {
                    r#match: r#match__.unwrap_or_default(),
                    segment_keys: segment_keys__.unwrap_or_default(),
                    reason: reason__.unwrap_or_default(),
                    variant_key: variant_key__.unwrap_or_default(),
                    variant_attachment: variant_attachment__.unwrap_or_default(),
                    request_id: request_id__.unwrap_or_default(),
                    request_duration_millis: request_duration_millis__.unwrap_or_default(),
                    timestamp: timestamp__,
                    flag_key: flag_key__.unwrap_or_default(),
                })
            }
        }
        deserializer.deserialize_struct("flipt.evaluation.VariantEvaluationResponse", FIELDS, GeneratedVisitor)
    }
}

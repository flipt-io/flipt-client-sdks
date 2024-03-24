pub struct StaticParser {
    data: String,
    namespace: String,
}

impl StaticParser {
    pub fn new(data: &str) -> Self {
        Self {
            data: data.to_string(),
        }
    }
}

impl Parser for StaticParser {
    fn parse(&self, namespace: &str) -> Result<source::Document, Error> {
        self.namespace = namespace.to_string();

        let document: source::Document = match serde_json::from_str(&self.data) {
            Ok(document) => document,
            Err(e) => return Err(Error::InvalidJSON(e)),
        };

        Ok(document)
    }
}


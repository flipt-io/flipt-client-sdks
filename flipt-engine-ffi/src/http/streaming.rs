// use reqwest::header::HeaderMap;

// use super::Authentication;

// pub struct HTTPStreamParser {
//     http_client: reqwest::Client,
//     http_url: String,
//     authentication: HeaderMap,
//     reference: Option<String>,
// }

// pub struct HTTPStreamParserBuilder {
//     http_url: String,
//     authentication: HeaderMap,
//     reference: Option<String>,
// }

// impl HTTPStreamParserBuilder {
//     pub fn new(http_url: &str) -> Self {
//         Self {
//             http_url: http_url.to_string(),
//             authentication: HeaderMap::new(),
//             reference: None,
//         }
//     }

//     pub fn authentication(mut self, authentication: Authentication) -> Self {
//         self.authentication = HeaderMap::from(authentication);
//         self
//     }

//     pub fn reference(mut self, reference: &str) -> Self {
//         self.reference = Some(reference.to_string());
//         self
//     }

//     pub fn build(self) -> HTTPStreamParser {
//         HTTPStreamParser {
//             http_client: reqwest::Client::new(),
//             http_url: self.http_url,
//             authentication: self.authentication,
//             reference: self.reference,
//         }
//     }
// }

// impl HTTPStreamParser {
//     fn url(&self, namespace: &str) -> String {
//         match &self.reference {
//             Some(reference) => {
//                 format!(
//                     "{}/internal/v1/evaluation/snapshots?namespace[]={}&reference={}",
//                     self.http_url, namespace, reference,
//                 )
//             }
//             None => {
//                 format!(
//                     "{}/internal/v1/evaluation/snapshots?namespace[]={}",
//                     self.http_url, namespace
//                 )
//             }
//         }
//     }
// }

// impl Parser for HTTPStreamParser {
//     async fn parse(&mut self, namespace: &str) -> Result<Option<source::Document>, Error> {
//         let mut headers = HeaderMap::new();
//         headers.insert(
//             reqwest::header::ACCEPT,
//             reqwest::header::HeaderValue::from_static("application/json"),
//         );
//         // version (or higher) that we can accept from the server
//         headers.insert(
//             "X-Flipt-Accept-Server-Version",
//             reqwest::header::HeaderValue::from_static("1.47.0"),
//         );

//         // add etag / if-none-match header if we have one
//         if let Some(etag) = &self.etag {
//             headers.insert(
//                 reqwest::header::IF_NONE_MATCH,
//                 reqwest::header::HeaderValue::from_str(etag).unwrap(),
//             );
//         }

//         for (key, value) in self.authentication.iter() {
//             headers.insert(key, value.clone());
//         }

//         let response = self
//             .http_client
//             .get(self.url(namespace))
//             .headers(headers)
//             .await?
//             .bytes_stream();

//         let mut buffer = Vec::new();

//         while let Some(chunk) = response.next().await {
//             for byte in chunk? {
//                 if byte == b'\n' {
//                     if let Ok(document) = serde_json::from_slice(&buffer) {
//                         return Ok(Some(document));
//                     }
//                     buffer.clear();
//                 } else {
//                     buffer.push(byte);
//                 }
//             }
//         }
//     }
// }

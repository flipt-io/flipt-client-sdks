use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("error engine null pointer")]
    NullPointer,
    #[error("error parsing json")]
    InvalidJSON(#[from] serde_json::Error),
    #[error("invalid request: {0}")]
    InvalidRequest(String),
    #[error("server error: {0}")]
    ServerError(String),
    #[error("unknown error: {0}")]
    Unknown(String),
}

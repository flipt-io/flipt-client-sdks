use thiserror::Error;

#[non_exhaustive]
#[derive(Error, Debug, Clone)]
pub enum Error {
    #[error("error parsing json: {0}")]
    InvalidJSON(String),
    #[error("invalid request: {0}")]
    InvalidRequest(String),
    #[error("server error: {0}")]
    Server(String),
    #[error("unknown error: {0}")]
    Unknown(String),
}

use serde::Serialize;
use thiserror::Error;

#[derive(Error, Debug, Clone, Serialize)]
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

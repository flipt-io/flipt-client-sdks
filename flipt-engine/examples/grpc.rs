// cargo run --example grpc

use protos::flipt::evaluation::data_service_client::DataServiceClient;
use protos::flipt::evaluation::EvaluationNamespaceSnapshotRequest;
use std::error::Error;
use tonic::{metadata::MetadataValue, transport::Channel, Request};

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    let channel = Channel::from_static("http://localhost:9000")
        .connect()
        .await?;

    let token: MetadataValue<_> = ("Bearer ".to_owned() + std::env!("FLIPT_AUTH_TOKEN")).parse()?;

    let mut client = DataServiceClient::with_interceptor(channel, move |mut req: Request<()>| {
        req.metadata_mut().insert("authorization", token.clone());
        Ok(req)
    });

    let out = client
        .evaluation_snapshot_namespace(EvaluationNamespaceSnapshotRequest {
            key: "default".into(),
        })
        .await?
        .into_inner();
    println!("variant key {:?}", out);
    Ok(())
}

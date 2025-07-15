@testable import FliptClient
import XCTest

class FliptClientTests: XCTestCase {
    var evaluationClient: FliptClient?
    var fliptUrl: String = ""
    var authToken: String = ""

    override func setUp() {
        super.setUp()

        guard let fliptUrl = ProcessInfo.processInfo.environment["FLIPT_URL"],
              let authToken = ProcessInfo.processInfo.environment["FLIPT_AUTH_TOKEN"]
        else {
            XCTFail("FLIPT_URL and FLIPT_AUTH_TOKEN must be set")
            return
        }

        do {
            var tlsConfig: TlsConfig?

            let caCertPath = ProcessInfo.processInfo.environment["FLIPT_CA_CERT_PATH"]
            if let caCertPath, !caCertPath.isEmpty {
                tlsConfig = try TlsConfig.builder()
                    .caCertFile(caCertPath)
                    .insecureSkipHostnameVerify(true)
                    .build()
            } else {
                // Fallback to insecure for local testing
                tlsConfig = try TlsConfig.builder()
                    .insecureSkipVerify(true)
                    .build()
            }

            evaluationClient = try FliptClient(
                namespace: "default",
                url: fliptUrl,
                authentication: .clientToken(authToken),
                tlsConfig: tlsConfig)
        } catch {
            XCTFail("Failed to initialize EvaluationClient: \(error)")
        }
    }

    override func tearDown() {
        evaluationClient?.close()
        evaluationClient = nil
        super.tearDown()
    }

    func testEmptyFlagKey() {
        do {
            _ = try evaluationClient?.evaluateBoolean(
                flagKey: "",
                entityID: "someentity",
                evalContext: ["fizz": "buzz"])
            XCTFail("Expected an error, but got none")
        } catch let error as FliptClient.ClientError {
            XCTAssertFalse(error.localizedDescription.isEmpty)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testEmptyEntityId() {
        do {
            _ = try evaluationClient?.evaluateBoolean(flagKey: "flag1", entityID: "", evalContext: ["fizz": "buzz"])
            XCTFail("Expected an error, but got none")
        } catch let error as FliptClient.ClientError {
            XCTAssertFalse(error.localizedDescription.isEmpty)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testInvalidAuthentication() {
        do {
            let client = try FliptClient(
                namespace: "default",
                url: fliptUrl,
                authentication: .clientToken("invalid"))

            _ = try client.evaluateVariant(flagKey: "flag1", entityID: "someentity", evalContext: ["fizz": "buzz"])
            XCTFail("Expected an error, but got none")
        } catch let error as FliptClient.ClientError {
            XCTAssertFalse(error.localizedDescription.isEmpty)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testVariant() {
        do {
            let variantResult = try evaluationClient?.evaluateVariant(
                flagKey: "flag1",
                entityID: "someentity",
                evalContext: ["fizz": "buzz"])
            guard let variantResult else {
                XCTFail("variantResult is nil")
                return
            }
            XCTAssertTrue(variantResult.match)
            XCTAssertEqual(variantResult.flag_key, "flag1")
            XCTAssertEqual(variantResult.reason, "MATCH_EVALUATION_REASON")
            XCTAssertTrue(variantResult.segment_keys.contains("segment1"))
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testBoolean() {
        do {
            let booleanResult = try evaluationClient?.evaluateBoolean(
                flagKey: "flag_boolean",
                entityID: "someentity",
                evalContext: ["fizz": "buzz"])
            guard let booleanResult else {
                XCTFail("booleanResult is nil")
                return
            }
            XCTAssertEqual(booleanResult.flag_key, "flag_boolean")
            XCTAssertTrue(booleanResult.enabled)
            XCTAssertEqual(booleanResult.reason, "MATCH_EVALUATION_REASON")
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testBatch() {
        do {
            let requests = [
                EvaluationRequest(flag_key: "flag1", entity_id: "someentity", context: ["fizz": "buzz"]),
                EvaluationRequest(flag_key: "flag_boolean", entity_id: "someentity", context: ["fizz": "buzz"]),
                EvaluationRequest(flag_key: "notfound", entity_id: "someentity", context: ["fizz": "buzz"])
            ]

            let batchResult = try evaluationClient?.evaluateBatch(requests: requests)
            guard let batchResult else {
                XCTFail("batchResult is nil")
                return
            }
            XCTAssertEqual(batchResult.responses.count, 3)

            let variantResult = batchResult.responses[0]
            XCTAssertEqual(variantResult.type, "VARIANT_EVALUATION_RESPONSE_TYPE")
            guard let variantResp = variantResult.variant_evaluation_response else {
                XCTFail("unexpected response type")
                return
            }
            XCTAssertTrue(variantResp.match)
            XCTAssertEqual(variantResp.flag_key, "flag1")
            XCTAssertEqual(variantResp.reason, "MATCH_EVALUATION_REASON")
            XCTAssertTrue(variantResp.segment_keys.contains("segment1"))

            let booleanResult = batchResult.responses[1]
            XCTAssertEqual(booleanResult.type, "BOOLEAN_EVALUATION_RESPONSE_TYPE")
            guard let boolResp = booleanResult.boolean_evaluation_response else {
                XCTFail("unexpected response type")
                return
            }
            XCTAssertEqual(boolResp.flag_key, "flag_boolean")
            XCTAssertTrue(boolResp.enabled)
            XCTAssertEqual(boolResp.reason, "MATCH_EVALUATION_REASON")

            let errorResult = batchResult.responses[2]
            XCTAssertEqual(errorResult.type, "ERROR_EVALUATION_RESPONSE_TYPE")
            guard let errResp = errorResult.error_evaluation_response else {
                XCTFail("unexpected response type")
                return
            }
            XCTAssertEqual(errResp.flag_key, "notfound")
            XCTAssertEqual(errResp.namespace_key, "default")
            XCTAssertEqual(errResp.reason, "NOT_FOUND_ERROR_EVALUATION_REASON")
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testListFlags() {
        do {
            let flags = try evaluationClient?.listFlags()
            guard let flags else {
                XCTFail("flags is nil")
                return
            }
            XCTAssertFalse(flags.isEmpty)
            XCTAssertEqual(flags.count, 2)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testVariantFailure() {
        do {
            _ = try evaluationClient?.evaluateVariant(
                flagKey: "nonexistent",
                entityID: "someentity",
                evalContext: ["fizz": "buzz"])
            XCTFail("Expected an error, but got none")
        } catch let error as FliptClient.ClientError {
            XCTAssertFalse(error.localizedDescription.isEmpty)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testSetGetSnapshotWithInvalidFliptURL() {
        do {
            let snapshot = try evaluationClient?.getSnapshot()
            XCTAssertNotNil(snapshot)

            let invalidFliptClient = try FliptClient(
                namespace: "default",
                url: "http://invalid.flipt.com",
                errorStrategy: .fallback,
                snapshot: snapshot)

            let context = ["fizz": "buzz"]

            for _ in 0 ..< 5 {
                // Variant evaluation
                let variant = try invalidFliptClient.evaluateVariant(
                    flagKey: "flag1",
                    entityID: "entity",
                    evalContext: context)
                XCTAssertEqual(variant.flag_key, "flag1")
                XCTAssertTrue(variant.match)
                XCTAssertEqual(variant.reason, "MATCH_EVALUATION_REASON")
                XCTAssertEqual(variant.variant_key, "variant1")
                XCTAssertTrue(variant.segment_keys.contains("segment1"))

                // Boolean evaluation
                let boolean = try invalidFliptClient.evaluateBoolean(
                    flagKey: "flag_boolean",
                    entityID: "entity",
                    evalContext: context)
                XCTAssertEqual(boolean.flag_key, "flag_boolean")
                XCTAssertTrue(boolean.enabled)
                XCTAssertEqual(boolean.reason, "MATCH_EVALUATION_REASON")

                // List flags
                let flags = try invalidFliptClient.listFlags()
                XCTAssertEqual(flags.count, 2)

                // Get snapshot
                let snapshot = try invalidFliptClient.getSnapshot()
                XCTAssertNotNil(snapshot)
            }

            invalidFliptClient.close()
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testTlsConfigSerialization() {
        do {
            let tlsConfig = try TlsConfig.builder()
                .caCertData("-----BEGIN CERTIFICATE-----")
                .insecureSkipVerify(true)
                .insecureSkipHostnameVerify(true)
                .clientCertData("-----BEGIN CERTIFICATE-----")
                .clientKeyData("-----BEGIN PRIVATE KEY-----")
                .build()

        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase

        do {
            let jsonData = try encoder.encode(tlsConfig)
            guard let jsonString = String(data: jsonData, encoding: .utf8) else {
                XCTFail("Failed to convert data to string")
                return
            }

            // Verify snake_case field names in JSON
            XCTAssertTrue(jsonString.contains("\"ca_cert_data\""))
            XCTAssertTrue(jsonString.contains("\"insecure_skip_verify\""))
            XCTAssertTrue(jsonString.contains("\"insecure_skip_hostname_verify\""))
            XCTAssertTrue(jsonString.contains("\"client_cert_data\""))
            XCTAssertTrue(jsonString.contains("\"client_key_data\""))

            // Verify values are correctly encoded
            XCTAssertTrue(jsonString.contains("\"-----BEGIN CERTIFICATE-----\""))
            XCTAssertTrue(jsonString.contains("true"))
            XCTAssertTrue(jsonString.contains("\"-----BEGIN PRIVATE KEY-----\""))

            // Verify file fields are not present (since we only used data fields)
            XCTAssertFalse(jsonString.contains("ca_cert_file"))
            XCTAssertFalse(jsonString.contains("client_cert_file"))
            XCTAssertFalse(jsonString.contains("client_key_file"))
        } catch {
            XCTFail("Failed to encode TlsConfig: \(error)")
        }
        } catch {
            XCTFail("Failed to create TlsConfig: \(error)")
        }
    }

    func testTlsConfigSerializationWithNilFields() {
        do {
            let tlsConfig = try TlsConfig.builder()
                .insecureSkipHostnameVerify(true)
                .build()

        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase

        do {
            let jsonData = try encoder.encode(tlsConfig)
            guard let jsonString = String(data: jsonData, encoding: .utf8) else {
                XCTFail("Failed to convert data to string")
                return
            }

            // Should only contain the field that was set
            XCTAssertTrue(jsonString.contains("\"insecure_skip_hostname_verify\":true"))

            // Should not contain null/nil fields
            XCTAssertFalse(jsonString.contains("ca_cert_file"))
            XCTAssertFalse(jsonString.contains("ca_cert_data"))
            XCTAssertFalse(jsonString.contains("insecure_skip_verify"))
            XCTAssertFalse(jsonString.contains("client_cert_file"))
            XCTAssertFalse(jsonString.contains("client_key_file"))
            XCTAssertFalse(jsonString.contains("client_cert_data"))
            XCTAssertFalse(jsonString.contains("client_key_data"))
        } catch {
            XCTFail("Failed to encode TlsConfig: \(error)")
        }
        } catch {
            XCTFail("Failed to create TlsConfig: \(error)")
        }
    }
}

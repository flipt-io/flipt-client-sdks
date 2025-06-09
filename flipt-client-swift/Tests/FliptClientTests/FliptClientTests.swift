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
            evaluationClient = try FliptClient(
                namespace: "default",
                url: fliptUrl,
                authentication: .clientToken(authToken))
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
            guard let variantResult = variantResult else {
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
            guard let booleanResult = booleanResult else {
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
            guard let batchResult = batchResult else {
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
            guard let flags = flags else {
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
}

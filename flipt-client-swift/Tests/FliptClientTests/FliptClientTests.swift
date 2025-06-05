@testable import FliptClient
import XCTest

class FliptClientTests: XCTestCase {
    var evaluationClient: FliptClient?
    var fliptUrl: String = ""
    var authToken: String = ""

    static var snapshot: String {
        base64EncodedResource(named: "snapshot")
    }

    private static func base64EncodedResource(named name: String) -> String {
        let bundle = Bundle(for: FliptClientTests.self)
        guard let url = bundle.url(forResource: name, withExtension: "json") else {
            XCTFail("Missing resource: \(name).json")
            return ""
        }
        guard let data = try? Data(contentsOf: url) else {
            XCTFail("Could not load data from: \(name).json")
            return ""
        }
        return data.base64EncodedString()
    }

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
            let variant = try evaluationClient?.evaluateVariant(
                flagKey: "flag1",
                entityID: "someentity",
                evalContext: ["fizz": "buzz"])
            XCTAssertTrue(variant?.match ?? false)
            XCTAssertEqual(variant?.flag_key, "flag1")
            XCTAssertEqual(variant?.reason, "MATCH_EVALUATION_REASON")
            XCTAssertTrue(variant?.segment_keys.contains("segment1") ?? false)
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testBoolean() {
        do {
            let boolean = try evaluationClient?.evaluateBoolean(
                flagKey: "flag_boolean",
                entityID: "someentity",
                evalContext: ["fizz": "buzz"])
            XCTAssertEqual(boolean?.flag_key, "flag_boolean")
            XCTAssertTrue(boolean?.enabled ?? false)
            XCTAssertEqual(boolean?.reason, "MATCH_EVALUATION_REASON")
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

            let batch = try evaluationClient?.evaluateBatch(requests: requests)
            XCTAssertEqual(batch?.responses.count, 3)

            let variant = batch?.responses[0]
            XCTAssertEqual(variant?.type, "VARIANT_EVALUATION_RESPONSE_TYPE")
            guard let variantResp = variant?.variant_evaluation_response else {
                XCTFail("unexpected response type")
                return
            }
            XCTAssertTrue(variantResp.match)
            XCTAssertEqual(variantResp.flag_key, "flag1")
            XCTAssertEqual(variantResp.reason, "MATCH_EVALUATION_REASON")
            XCTAssertTrue(variantResp.segment_keys.contains("segment1"))

            let boolean = batch?.responses[1]
            XCTAssertEqual(boolean?.type, "BOOLEAN_EVALUATION_RESPONSE_TYPE")
            guard let boolResp = boolean?.boolean_evaluation_response else {
                XCTFail("unexpected response type")
                return
            }
            XCTAssertEqual(boolResp.flag_key, "flag_boolean")
            XCTAssertTrue(boolResp.enabled)
            XCTAssertEqual(boolResp.reason, "MATCH_EVALUATION_REASON")

            let errorResponse = batch?.responses[2]
            XCTAssertEqual(errorResponse?.type, "ERROR_EVALUATION_RESPONSE_TYPE")
            guard let errResp = errorResponse?.error_evaluation_response else {
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
            XCTAssertFalse(flags?.isEmpty ?? true)
            XCTAssertEqual(flags?.count, 2)
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
            let invalidFliptClient = try FliptClient(
                namespace: "default",
                url: "http://invalid.flipt.com",
                errorStrategy: .fallback,
                snapshot: FliptClientTests.snapshot)

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

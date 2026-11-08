import FliptEngineFFI
import Foundation

public class FliptClient {
    private var engine: UnsafeMutableRawPointer?
    private var environment: String = "default"
    private var namespace: String = "default"
    private var url: String = "http://localhost:8080"
    private var authentication: Authentication?
    private var reference: String?
    private var requestTimeout: Int = 0
    private var updateInterval: Int = 120
    private var fetchMode: FetchMode = .polling
    private var errorStrategy: ErrorStrategy = .fail
    private var snapshot: String?
    private var tlsConfig: TlsConfig?
    private var stateUpdates: StateUpdates?

    public init(
        environment: String? = nil,
        namespace: String? = nil,
        url: String? = nil,
        authentication: Authentication? = nil,
        reference: String? = nil,
        requestTimeout: Duration? = nil,
        updateInterval: Duration? = nil,
        fetchMode: FetchMode = .polling,
        errorStrategy: ErrorStrategy = .fail,
        snapshot: String? = nil,
        tlsConfig: TlsConfig? = nil,
        onStateUpdate: (@Sendable () -> Void)? = nil) throws
    {
        self.environment = environment ?? "default"
        self.namespace = namespace ?? "default"
        self.url = url ?? "http://localhost:8080"
        self.authentication = authentication
        self.requestTimeout = Int(requestTimeout?.components.seconds ?? 0)
        self.updateInterval = Int(updateInterval?.components.seconds ?? 120)
        self.reference = reference
        self.fetchMode = fetchMode
        self.errorStrategy = errorStrategy
        self.snapshot = snapshot
        self.tlsConfig = tlsConfig

        let clientOptions = ClientOptions(
            environment: self.environment,
            namespace: self.namespace,
            url: self.url,
            authentication: self.authentication,
            requestTimeout: self.requestTimeout,
            updateInterval: self.updateInterval,
            reference: self.reference,
            fetchMode: self.fetchMode,
            errorStrategy: self.errorStrategy,
            snapshot: self.snapshot,
            tlsConfig: self.tlsConfig)

        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        guard let jsonData = try? encoder.encode(clientOptions),
              let jsonStr = String(data: jsonData, encoding: .utf8)
        else {
            throw ClientError.invalidOptions
        }

        let clientOptionsCString = strdup(jsonStr)

        defer {
            free(clientOptionsCString)
        }

        engine = initialize_engine(clientOptionsCString)
        stateUpdates = StateUpdates(engine: engine)

        if let onStateUpdate {
            do {
                try addStateUpdateListener(onStateUpdate)
            } catch {
                // The caller never gets a client to close, so tear the engine down here.
                close()
                throw error
            }
        }
    }

    deinit {
        close()
    }

    /// Closes the client and releases the engine. When called from a thread other than the state
    /// update thread, waits for a running listener to return first. Closing twice is a no-op.
    public func close() {
        // Only the first caller tears down the engine, after any running listener has returned.
        // Later callers wait for that listener too, then return.
        guard let stateUpdates, stateUpdates.close() else {
            return
        }
        // Clear the pointer first, so later calls see a closed client rather than a freed engine.
        if let engine {
            self.engine = nil
            destroy_engine(engine)
        }
    }

    /// Adds a listener that is called after each change to flag state. Listeners can be added at any
    /// time. Read the flags you care about once after adding a listener: it is called for changes
    /// applied after it was added, and may occasionally see one extra call.
    ///
    /// Listeners run in the order they were added, on a single thread named `flipt-state-update`.
    /// Updates that arrive while they run are coalesced into a single follow-up call. A listener may
    /// evaluate flags, add or remove listeners, or call `close()`. `close()` from another thread
    /// waits for a running listener to return, so a listener must not block on anything held by the
    /// thread that closes the client.
    ///
    /// A listener that captures the client strongly keeps it alive until `close()` is called, so
    /// capture it weakly or close the client explicitly.
    ///
    /// - Returns: A subscription that removes the listener when closed.
    /// - Throws: `ClientError.evaluationFailed` if the client is closed or the engine cannot
    ///   subscribe to updates.
    @discardableResult
    public func addStateUpdateListener(_ listener: @escaping @Sendable () -> Void) throws -> StateUpdateSubscription {
        guard let stateUpdates else {
            throw ClientError.evaluationFailed(message: StateUpdates.closedMessage)
        }
        return try stateUpdates.add(listener)
    }

    public func evaluateVariant(
        flagKey: String,
        entityID: String,
        evalContext: [String: String]) throws -> VariantEvaluationResponse
    {
        if flagKey.isEmpty {
            throw ClientError.invalidRequest
        }
        if entityID.isEmpty {
            throw ClientError.invalidRequest
        }

        let evaluationRequest = EvaluationRequest(
            flag_key: flagKey,
            entity_id: entityID,
            context: evalContext)

        guard let requestData = try? JSONEncoder().encode(evaluationRequest) else {
            throw ClientError.invalidRequest
        }

        let requestCString = strdup(String(data: requestData, encoding: .utf8))

        let variantResponse = evaluate_variant(engine, requestCString)
        free(requestCString)

        guard let response = variantResponse else {
            throw ClientError.evaluationFailed(message: "No response from engine")
        }

        let responseString = String(cString: response)
        destroy_string(UnsafeMutablePointer(mutating: response))

        if responseString.isEmpty {
            throw ClientError.evaluationFailed(message: "Empty response from engine")
        }

        guard let variantResult = try? JSONDecoder().decode(VariantResult.self, from: Data(responseString.utf8)) else {
            throw ClientError.parsingError
        }

        if variantResult.status != "success" {
            throw ClientError.evaluationFailed(message: variantResult.error_message ?? "Unknown error")
        }

        guard let result = variantResult.result else {
            throw ClientError.evaluationFailed(message: "missing result")
        }

        return result
    }

    public func evaluateBoolean(
        flagKey: String,
        entityID: String,
        evalContext: [String: String]) throws -> BooleanEvaluationResponse
    {
        if flagKey.isEmpty {
            throw ClientError.invalidRequest
        }
        if entityID.isEmpty {
            throw ClientError.invalidRequest
        }

        let evaluationRequest = EvaluationRequest(
            flag_key: flagKey,
            entity_id: entityID,
            context: evalContext)

        guard let requestData = try? JSONEncoder().encode(evaluationRequest) else {
            throw ClientError.invalidRequest
        }

        let requestCString = strdup(String(data: requestData, encoding: .utf8))

        let booleanResponse = evaluate_boolean(engine, requestCString)
        free(requestCString)

        guard let response = booleanResponse else {
            throw ClientError.evaluationFailed(message: "No response from engine")
        }

        let responseString = String(cString: response)
        destroy_string(UnsafeMutablePointer(mutating: response))

        if responseString.isEmpty {
            throw ClientError.evaluationFailed(message: "Empty response from engine")
        }

        guard let booleanResult = try? JSONDecoder().decode(BooleanResult.self, from: Data(responseString.utf8)) else {
            throw ClientError.parsingError
        }

        if booleanResult.status != "success" {
            throw ClientError.evaluationFailed(message: booleanResult.error_message ?? "Unknown error")
        }

        guard let result = booleanResult.result else {
            throw ClientError.evaluationFailed(message: "missing result")
        }

        return result
    }

    public func listFlags() throws -> [Flag] {
        let flagsResponse = list_flags(engine)

        guard let response = flagsResponse else {
            throw ClientError.evaluationFailed(message: "No response from engine")
        }
        let responseString = String(cString: response)
        destroy_string(UnsafeMutablePointer(mutating: response))

        if responseString.isEmpty {
            throw ClientError.evaluationFailed(message: "Empty response from engine")
        }

        guard let listFlagsResult = try? JSONDecoder().decode(ListFlagsResult.self, from: Data(responseString.utf8))
        else {
            throw ClientError.parsingError
        }

        if listFlagsResult.status != "success" {
            throw ClientError.evaluationFailed(message: listFlagsResult.error_message ?? "Unknown error")
        }

        guard let result = listFlagsResult.result else {
            throw ClientError.evaluationFailed(message: "missing result")
        }

        return result
    }

    public func evaluateBatch(requests: [EvaluationRequest]) throws -> BatchEvaluationResponse {
        guard let requestsData = try? JSONEncoder().encode(requests) else {
            throw ClientError.invalidRequest
        }

        let requestCString = strdup(String(data: requestsData, encoding: .utf8))

        let batchResponse = evaluate_batch(engine, requestCString)
        free(requestCString)

        guard let response = batchResponse else {
            throw ClientError.evaluationFailed(message: "No response from engine")
        }

        let responseString = String(cString: response)
        destroy_string(UnsafeMutablePointer(mutating: response))

        if responseString.isEmpty {
            throw ClientError.evaluationFailed(message: "Empty response from engine")
        }

        guard let batchResult = try? JSONDecoder().decode(BatchResult.self, from: Data(responseString.utf8)) else {
            throw ClientError.parsingError
        }

        if batchResult.status != "success" {
            throw ClientError.evaluationFailed(message: batchResult.error_message ?? "Unknown error")
        }

        guard let result = batchResult.result else {
            throw ClientError.evaluationFailed(message: "missing result")
        }

        return result
    }

    public func getSnapshot() throws -> String {
        guard let snapshotCString = get_snapshot(engine) else {
            throw ClientError.evaluationFailed(message: "No response from engine")
        }
        let snapshot = String(cString: snapshotCString)
        destroy_string(UnsafeMutablePointer(mutating: snapshotCString))
        return snapshot
    }

    public enum ClientError: Error {
        case invalidOptions
        case invalidRequest
        case evaluationFailed(message: String)
        case parsingError
    }
}

public enum Authentication: Encodable {
    case clientToken(String)
    case jwtToken(String)

    /** Custom Codable logic to encode/decode based on the case */
    private enum CodingKeys: String, CodingKey {
        case client_token
        case jwt_token
    }

    public enum AuthenticationType: String, Codable {
        case clientToken, jwtToken
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        switch self {
        case let .clientToken(token):
            try container.encode(token, forKey: .client_token)
        case let .jwtToken(token):
            try container.encode(token, forKey: .jwt_token)
        }
    }
}

public struct EvaluationRequest: Codable {
    public let flag_key: String
    public let entity_id: String
    public let context: [String: String]

    public init(
        flag_key: String,
        entity_id: String,
        context: [String: String])
    {
        self.flag_key = flag_key
        self.entity_id = entity_id
        self.context = context
    }
}

public struct ClientTokenAuthentication: Codable {
    public let token: String
}

public struct JWTAuthentication: Codable {
    public let token: String
}

public enum FetchMode: String, Codable {
    case streaming
    case polling
}

public enum ErrorStrategy: String, Codable {
    case fail
    case fallback
}

public struct ClientOptions<T: Encodable>: Encodable {
    public let environment: String
    public let namespace: String
    public let url: String
    public let authentication: T?
    public let requestTimeout: Int
    public let updateInterval: Int
    public let reference: String?
    public let fetchMode: FetchMode
    public let errorStrategy: ErrorStrategy
    public let snapshot: String?
    public let tlsConfig: TlsConfig?
}

public struct Flag: Codable {
    public let key: String
    public let enabled: Bool
    public let type: String
}

public struct VariantEvaluationResponse: Codable {
    public let match: Bool
    public let segment_keys: [String]
    public let reason: String
    public let flag_key: String
    public let variant_key: String
    public let variant_attachment: String?
    public let request_duration_millis: Double
    public let timestamp: String
}

public struct BooleanEvaluationResponse: Codable {
    public let enabled: Bool
    public let flag_key: String
    public let reason: String
    public let request_duration_millis: Double
    public let timestamp: String
    public let segment_keys: [String]
}

public struct ErrorEvaluationResponse: Codable {
    public let flag_key: String
    public let namespace_key: String
    public let reason: String
}

public struct BatchEvaluationResponse: Codable {
    public let responses: [Response]
    public let request_duration_millis: Double
}

public struct Response: Codable {
    public let type: String
    public let variant_evaluation_response: VariantEvaluationResponse?
    public let boolean_evaluation_response: BooleanEvaluationResponse?
    public let error_evaluation_response: ErrorEvaluationResponse?
}

struct Result<R: Codable>: Codable {
    let status: String
    let result: R?
    let error_message: String?
}

// Specific result types
typealias VariantResult = Result<VariantEvaluationResponse>
typealias BooleanResult = Result<BooleanEvaluationResponse>
typealias BatchResult = Result<BatchEvaluationResponse>
typealias ListFlagsResult = Result<[Flag]>

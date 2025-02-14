import Foundation
import FliptEngineFFI

public class FliptClient {
    private var engine: UnsafeMutableRawPointer?
    private var namespace: String = "default"
    private var url: String = ""
    private var authentication: Authentication?
    private var ref: String = ""
    private var updateInterval: Int = 0
    private var fetchMode: FetchMode = .polling
    private var errorStrategy: ErrorStrategy = .fail

    public init(namespace: String = "default",
         url: String = "",
         authentication: Authentication? = nil,
         ref: String = "",
         updateInterval: Int = 120,
         fetchMode: FetchMode = .polling,
         errorStrategy: ErrorStrategy = .fail) throws {
        self.namespace = namespace
        self.url = url
        self.authentication = authentication
        self.ref = ref
        self.updateInterval = updateInterval
        self.fetchMode = fetchMode
        self.errorStrategy = errorStrategy

        let clientOptions = ClientOptions(
            url: url,
            authentication: authentication,
            updateInterval: updateInterval,
            reference: ref,
            fetchMode: fetchMode,
            errorStrategy: errorStrategy
        )

        guard let jsonData = try? JSONEncoder().encode(clientOptions) else {
            throw ClientError.invalidOptions
        }

        let jsonStr = String(data: jsonData, encoding: .utf8)
        let namespaceCString = strdup(namespace)
        let clientOptionsCString = strdup(jsonStr)

        engine = initialize_engine(namespaceCString, clientOptionsCString)

        free(namespaceCString)
        free(clientOptionsCString)
    }

    deinit {
        close()
    }

    public func close() {
        if let engine = engine {
            destroy_engine(engine)
            self.engine = nil
        }
    }

    public func evaluateVariant(flagKey: String, entityID: String, evalContext: [String: String]) throws -> VariantEvaluationResponse {
        if flagKey.isEmpty {
            throw ClientError.invalidRequest
        }
        if entityID.isEmpty {
            throw ClientError.invalidRequest
        }

        let evaluationRequest = EvaluationRequest(
            flag_key: flagKey,
            entity_id: entityID,
            context: evalContext
        )

        guard let requestData = try? JSONEncoder().encode(evaluationRequest) else {
            throw ClientError.invalidRequest
        }

        let requestCString = strdup(String(data: requestData, encoding: .utf8))

        let variantResponse = evaluate_variant(engine, requestCString)
        free(requestCString)

        let responseString = String(cString: variantResponse!)
        destroy_string(UnsafeMutablePointer(mutating: variantResponse))

        do {
            let variantResult = try JSONDecoder().decode(VariantResult.self, from: Data(responseString.utf8))
            // Use variantResult here
            if variantResult.status != "success" {
                throw ClientError.evaluationFailed(message: variantResult.error_message ?? "Unknown error")
            }
            guard let result = variantResult.result else {
                throw ClientError.evaluationFailed(message: "missing result")
            }

            return result
        } catch {
            throw ClientError.parsingError
        }

    }

    public func evaluateBoolean(flagKey: String, entityID: String, evalContext: [String: String]) throws -> BooleanEvaluationResponse {
        if flagKey.isEmpty {
            throw ClientError.invalidRequest
        }
        if entityID.isEmpty {
            throw ClientError.invalidRequest
        }

        let evaluationRequest = EvaluationRequest(
            flag_key: flagKey,
            entity_id: entityID,
            context: evalContext
        )

        guard let requestData = try? JSONEncoder().encode(evaluationRequest) else {
            throw ClientError.invalidRequest
        }

        let requestCString = strdup(String(data: requestData, encoding: .utf8))

        let booleanResponse = evaluate_boolean(engine, requestCString)
        free(requestCString)

        let responseString = String(cString: booleanResponse!)
        destroy_string(UnsafeMutablePointer(mutating: booleanResponse))

        guard let booleanResult = try? JSONDecoder().decode(BooleanResult.self, from: Data(responseString.utf8)) else {
            throw ClientError.parsingError
        }

        if booleanResult.status != "success" {
            throw ClientError.evaluationFailed(message: booleanResult.error_message ?? "Unknown error")
        }

        return booleanResult.result!
    }

    public func listFlags() throws -> [Flag] {
        let flagsResponse = list_flags(engine)

        let responseString = String(cString: flagsResponse!)
        destroy_string(UnsafeMutablePointer(mutating: flagsResponse))

        guard let listFlagsResult = try? JSONDecoder().decode(ListFlagsResult.self, from: Data(responseString.utf8)) else {
            throw ClientError.parsingError
        }

        if listFlagsResult.status != "success" {
            throw ClientError.evaluationFailed(message: listFlagsResult.error_message ?? "Unknown error")
        }

        return listFlagsResult.result!
    }

    public func evaluateBatch(requests: [EvaluationRequest]) throws -> BatchEvaluationResponse {
        guard let requestsData = try? JSONEncoder().encode(requests) else {
            throw ClientError.invalidRequest
        }

        let requestCString = strdup(String(data: requestsData, encoding: .utf8))

        let batchResponse = evaluate_batch(engine, requestCString)
        free(requestCString)

        let responseString = String(cString: batchResponse!)
        destroy_string(UnsafeMutablePointer(mutating: batchResponse))

        guard let batchResult = try? JSONDecoder().decode(BatchResult.self, from: Data(responseString.utf8)) else {
            throw ClientError.parsingError
        }

        if batchResult.status != "success" {
            throw ClientError.evaluationFailed(message: batchResult.error_message ?? "Unknown error")
        }

        return batchResult.result!
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

    // Custom Codable logic to encode/decode based on the case
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
        case .clientToken(let token):
            try container.encode(token, forKey: .client_token)
        case .jwtToken(let token):
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
        context: [String: String]
    ) {
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
    public let url: String
    public let authentication: T?
    public let updateInterval: Int
    public let reference: String
    public let fetchMode: FetchMode
    public let errorStrategy: ErrorStrategy
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

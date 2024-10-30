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

    public init(namespace: String = "default",
         url: String = "",
         authentication: Authentication? = nil,
         ref: String = "",
         updateInterval: Int = 120,
         fetchMode: FetchMode = .polling) throws {
        self.namespace = namespace
        self.url = url
        self.authentication = authentication
        self.ref = ref
        self.updateInterval = updateInterval
        self.fetchMode = fetchMode

        let clientOptions = ClientOptions(
            url: url,
            authentication: authentication,
            updateInterval: updateInterval,
            reference: ref,
            fetchMode: fetchMode
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

    enum AuthenticationType: String, Codable {
        case clientToken, jwtToken
    }

    func encode(to encoder: Encoder) throws {
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
    let flag_key: String
    let entity_id: String
    let context: [String: String]
}

public struct ClientTokenAuthentication: Codable {
    let token: String
}

public struct JWTAuthentication: Codable {
    let token: String
}

public enum FetchMode: String, Codable {
    case streaming
    case polling
}

public struct ClientOptions<T: Encodable>: Encodable {
    let url: String
    let authentication: T?
    let updateInterval: Int
    let reference: String
    let fetchMode: FetchMode
}

public struct Flag: Codable {
    let key: String
    let enabled: Bool
    let type: String
}

public struct VariantEvaluationResponse: Codable {
    let match: Bool
    let segment_keys: [String]
    let reason: String
    let flag_key: String
    let variant_key: String
    let variant_attachment: String?
    let request_duration_millis: Double
    let timestamp: String
}

public struct BooleanEvaluationResponse: Codable {
    let enabled: Bool
    let flag_key: String
    let reason: String
    let request_duration_millis: Double
    let timestamp: String
}

public struct ErrorEvaluationResponse: Codable {
    let flag_key: String
    let namespace_key: String
    let reason: String
}

public struct BatchEvaluationResponse: Codable {
    let responses: [Response]
    let request_duration_millis: Double
}

public struct Response: Codable {
    let type: String
    let variant_evaluation_response: VariantEvaluationResponse?
    let boolean_evaluation_response: BooleanEvaluationResponse?
    let error_evaluation_response: ErrorEvaluationResponse?
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

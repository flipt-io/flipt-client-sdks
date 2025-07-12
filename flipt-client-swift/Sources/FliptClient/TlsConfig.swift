import Foundation

public struct TlsConfig: Codable {
    public let caCertFile: String?
    public let caCertData: String?
    public let insecureSkipVerify: Bool?
    public let insecureSkipHostnameVerify: Bool?
    public let clientCertFile: String?
    public let clientKeyFile: String?
    public let clientCertData: String?
    public let clientKeyData: String?
    
    public init(
        caCertFile: String? = nil,
        caCertData: String? = nil,
        insecureSkipVerify: Bool? = nil,
        insecureSkipHostnameVerify: Bool? = nil,
        clientCertFile: String? = nil,
        clientKeyFile: String? = nil,
        clientCertData: String? = nil,
        clientKeyData: String? = nil
    ) {
        self.caCertFile = caCertFile
        self.caCertData = caCertData
        self.insecureSkipVerify = insecureSkipVerify
        self.insecureSkipHostnameVerify = insecureSkipHostnameVerify
        self.clientCertFile = clientCertFile
        self.clientKeyFile = clientKeyFile
        self.clientCertData = clientCertData
        self.clientKeyData = clientKeyData
    }
    
    public class Builder {
        private var caCertFile: String?
        private var caCertData: String?
        private var insecureSkipVerify: Bool?
        private var insecureSkipHostnameVerify: Bool?
        private var clientCertFile: String?
        private var clientKeyFile: String?
        private var clientCertData: String?
        private var clientKeyData: String?
        
        public init() {}
        
        public func caCertFile(_ value: String) -> Builder {
            self.caCertFile = value
            return self
        }
        
        public func caCertData(_ value: String) -> Builder {
            self.caCertData = value
            return self
        }
        
        public func insecureSkipVerify(_ value: Bool) -> Builder {
            self.insecureSkipVerify = value
            return self
        }
        
        public func insecureSkipHostnameVerify(_ value: Bool) -> Builder {
            self.insecureSkipHostnameVerify = value
            return self
        }
        
        public func clientCertFile(_ value: String) -> Builder {
            self.clientCertFile = value
            return self
        }
        
        public func clientKeyFile(_ value: String) -> Builder {
            self.clientKeyFile = value
            return self
        }
        
        public func clientCertData(_ value: String) -> Builder {
            self.clientCertData = value
            return self
        }
        
        public func clientKeyData(_ value: String) -> Builder {
            self.clientKeyData = value
            return self
        }
        
        public func build() throws -> TlsConfig {
            // Validate certificate files exist if specified
            if let caCertFile = caCertFile, !caCertFile.isEmpty {
                if !FileManager.default.fileExists(atPath: caCertFile) {
                    throw TlsConfigError.fileNotFound("CA certificate file does not exist: \(caCertFile)")
                }
            }
            
            if let clientCertFile = clientCertFile, !clientCertFile.isEmpty {
                if !FileManager.default.fileExists(atPath: clientCertFile) {
                    throw TlsConfigError.fileNotFound("Client certificate file does not exist: \(clientCertFile)")
                }
            }
            
            if let clientKeyFile = clientKeyFile, !clientKeyFile.isEmpty {
                if !FileManager.default.fileExists(atPath: clientKeyFile) {
                    throw TlsConfigError.fileNotFound("Client key file does not exist: \(clientKeyFile)")
                }
            }
            
            return TlsConfig(
                caCertFile: caCertFile,
                caCertData: caCertData,
                insecureSkipVerify: insecureSkipVerify,
                insecureSkipHostnameVerify: insecureSkipHostnameVerify,
                clientCertFile: clientCertFile,
                clientKeyFile: clientKeyFile,
                clientCertData: clientCertData,
                clientKeyData: clientKeyData
            )
        }
    }
    
    public static func builder() -> Builder {
        return Builder()
    }
}

public enum TlsConfigError: Error {
    case fileNotFound(String)
    case invalidConfiguration(String)
}
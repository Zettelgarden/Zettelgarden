import Foundation

public struct AppConfiguration: Codable, Equatable {
    public var baseURL: URL?
    public var apiToken: String
    public var defaultDestination: DestinationType

    public init(baseURL: URL?, apiToken: String, defaultDestination: DestinationType) {
        self.baseURL = baseURL
        self.apiToken = apiToken
        self.defaultDestination = defaultDestination
    }
}

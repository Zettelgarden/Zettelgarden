import Foundation

public struct ShareRequest: Codable {
    public var title: String
    public var body: String
    public var sourceURL: URL?
    public var destination: DestinationType

    public init(title: String, body: String, sourceURL: URL?, destination: DestinationType) {
        self.title = title
        self.body = body
        self.sourceURL = sourceURL
        self.destination = destination
    }
}

public enum ZettelgardenAPIError: Error, LocalizedError {
    case missingConfiguration
    case network(Error)
    case invalidResponse
    case server(message: String)

    public var errorDescription: String? {
        switch self {
        case .missingConfiguration:
            return "Configure the API base URL and token in the host app before sharing."
        case .network(let error):
            return error.localizedDescription
        case .invalidResponse:
            return "The server returned an unexpected response."
        case .server(let message):
            return message
        }
    }
}

public final class ZettelgardenAPI {
    private let urlSession: URLSession
    private let configurationStore: AppConfigurationStore

    public init(configurationStore: AppConfigurationStore = UserDefaultsConfigurationStore(),
                urlSession: URLSession = .shared) {
        self.configurationStore = configurationStore
        self.urlSession = urlSession
    }

    public func submit(request: ShareRequest) async throws {
        let configuration = configurationStore.loadConfiguration()
        guard let baseURL = configuration.baseURL, !configuration.apiToken.isEmpty else {
            throw ZettelgardenAPIError.missingConfiguration
        }

        let endpoint = baseURL.appendingPathComponent("api/v1/").appendingPathComponent(request.destination.pathComponent)
        var urlRequest = URLRequest(url: endpoint)
        urlRequest.httpMethod = "POST"
        urlRequest.addValue("application/json", forHTTPHeaderField: "Content-Type")
        urlRequest.addValue("Bearer \(configuration.apiToken)", forHTTPHeaderField: "Authorization")

        let payload = SharePayload(title: request.title, body: request.body, sourceURL: request.sourceURL?.absoluteString)
        urlRequest.httpBody = try JSONEncoder().encode(payload)

        do {
            let (data, response) = try await urlSession.data(for: urlRequest)
            guard let httpResponse = response as? HTTPURLResponse else {
                throw ZettelgardenAPIError.invalidResponse
            }

            switch httpResponse.statusCode {
            case 200..<300:
                return
            default:
                let serverMessage = (try? JSONDecoder().decode(ServerError.self, from: data))?.message
                throw ZettelgardenAPIError.server(message: serverMessage ?? "Server returned status code \(httpResponse.statusCode)")
            }
        } catch {
            throw ZettelgardenAPIError.network(error)
        }
    }
}

private struct SharePayload: Codable {
    var title: String
    var body: String
    var sourceURL: String?
}

private struct ServerError: Codable {
    var message: String
}

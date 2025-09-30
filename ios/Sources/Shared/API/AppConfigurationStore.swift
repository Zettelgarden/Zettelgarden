import Foundation

public protocol AppConfigurationStore {
    func loadConfiguration() -> AppConfiguration
    func save(configuration: AppConfiguration)
}

public final class UserDefaultsConfigurationStore: AppConfigurationStore {
    private let suiteName: String
    private let configurationKey = "zettelgarden.configuration"

    public init(appGroupIdentifier: String = AppGroups.primary.identifier) {
        self.suiteName = appGroupIdentifier
    }

    public func loadConfiguration() -> AppConfiguration {
        guard let defaults = UserDefaults(suiteName: suiteName),
              let data = defaults.data(forKey: configurationKey),
              let configuration = try? JSONDecoder().decode(AppConfiguration.self, from: data) else {
            return .empty
        }
        return configuration
    }

    public func save(configuration: AppConfiguration) {
        guard let defaults = UserDefaults(suiteName: suiteName),
              let data = try? JSONEncoder().encode(configuration) else {
            return
        }
        defaults.set(data, forKey: configurationKey)
    }
}

public final class InMemoryConfigurationStore: AppConfigurationStore {
    private var configuration: AppConfiguration

    public init(configuration: AppConfiguration = .empty) {
        self.configuration = configuration
    }

    public func loadConfiguration() -> AppConfiguration {
        configuration
    }

    public func save(configuration: AppConfiguration) {
        self.configuration = configuration
    }
}

import Combine
import Foundation
import ZettelgardenShared

final class AppModel: ObservableObject {
    @Published var configuration: AppConfiguration

    init(store: AppConfigurationStore = UserDefaultsConfigurationStore()) {
        self.store = store
        self.configuration = store.loadConfiguration()
    }

    func save(configuration: AppConfiguration) {
        store.save(configuration: configuration)
        self.configuration = configuration
    }

    private let store: AppConfigurationStore
}

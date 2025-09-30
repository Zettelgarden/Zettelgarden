import SwiftUI
import ZettelgardenShared

@main
struct ZettelgardenApp: App {
    @StateObject private var appModel = AppModel()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(appModel)
        }
    }
}

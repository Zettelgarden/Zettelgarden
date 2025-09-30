import Foundation
import Combine
import ZettelgardenShared

@MainActor
final class ShareExtensionViewModel: ObservableObject {
    enum State: Equatable {
        case composing
        case submitting
        case completed
        case failed(String)
    }

    @Published var shareItem: ShareItem
    @Published var destination: DestinationType
    @Published private(set) var state: State = .composing

    private let api: ZettelgardenAPI

    init(initialItem: ShareItem = ShareItem(),
         defaultDestination: DestinationType = .card,
         api: ZettelgardenAPI = ZettelgardenAPI()) {
        self.shareItem = initialItem
        self.destination = defaultDestination
        self.api = api
    }

    func submit() async {
        guard !shareItem.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            state = .failed("Title is required before sharing.")
            return
        }

        state = .submitting
        let request = ShareRequest(title: shareItem.title,
                                   body: shareItem.body,
                                   sourceURL: shareItem.url,
                                   destination: destination)
        do {
            try await api.submit(request: request)
            state = .completed
        } catch {
            let message = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
            state = .failed(message)
        }
    }

    func resetError() {
        if case .failed = state {
            state = .composing
        }
    }
}

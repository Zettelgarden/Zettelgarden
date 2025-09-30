import UIKit
import SwiftUI
import UniformTypeIdentifiers
import ZettelgardenShared

final class ShareViewController: UIViewController {
    private var viewModel: ShareExtensionViewModel!

    override func viewDidLoad() {
        super.viewDidLoad()
        configureViewModel()
        embedSwiftUIView()
        Task {
            await populateInitialContent()
        }
    }

    private func configureViewModel() {
        let store = UserDefaultsConfigurationStore()
        let configuration = store.loadConfiguration()
        viewModel = ShareExtensionViewModel(defaultDestination: configuration.defaultDestination)
    }

    private func embedSwiftUIView() {
        let controller = UIHostingController(rootView: ShareComposerView(viewModel: viewModel) { [weak self] in
            self?.handleDismiss()
        })
        addChild(controller)
        controller.view.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(controller.view)
        NSLayoutConstraint.activate([
            controller.view.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            controller.view.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            controller.view.topAnchor.constraint(equalTo: view.topAnchor),
            controller.view.bottomAnchor.constraint(equalTo: view.bottomAnchor)
        ])
        controller.didMove(toParent: self)
    }

    private func handleDismiss() {
        extensionContext?.completeRequest(returningItems: nil)
    }

    private func populateInitialContent() async {
        guard let context = extensionContext else { return }
        let shareItem = await extractShareItem(from: context)
        await MainActor.run {
            viewModel.shareItem = shareItem
            if viewModel.shareItem.title.isEmpty, !viewModel.shareItem.body.isEmpty {
                viewModel.shareItem.title = viewModel.shareItem.body.components(separatedBy: "\n").first ?? ""
            }
        }
    }

    private func extractShareItem(from context: NSExtensionContext) async -> ShareItem {
        let extensionItems = context.inputItems.compactMap { $0 as? NSExtensionItem }
        var collected = ShareItem()

        if let primaryText = extensionItems.compactMap({ $0.attributedContentText?.string }).first, !primaryText.isEmpty {
            collected.title = primaryText
        }

        let providers = extensionItems.flatMap { $0.attachments ?? [] }

        for provider in providers {
            if provider.hasItemConformingToTypeIdentifier(UTType.url.identifier), collected.url == nil {
                if let url = await loadURL(from: provider) {
                    collected.url = url
                }
            }
            if provider.hasItemConformingToTypeIdentifier(UTType.plainText.identifier) {
                if let text = await loadText(from: provider) {
                    if collected.body.isEmpty {
                        if collected.title.isEmpty {
                            collected.title = text
                        } else {
                            collected.body = text
                        }
                    } else {
                        collected.body.append("\n\n\(text)")
                    }
                }
            }
        }
        return collected
    }

    private func loadURL(from provider: NSItemProvider) async -> URL? {
        await withCheckedContinuation { continuation in
            provider.loadItem(forTypeIdentifier: UTType.url.identifier, options: nil) { item, _ in
                switch item {
                case let url as URL:
                    continuation.resume(returning: url)
                case let data as Data:
                    continuation.resume(returning: URL(dataRepresentation: data, relativeTo: nil))
                default:
                    continuation.resume(returning: nil)
                }
            }
        }
    }

    private func loadText(from provider: NSItemProvider) async -> String? {
        await withCheckedContinuation { continuation in
            provider.loadItem(forTypeIdentifier: UTType.plainText.identifier, options: nil) { item, _ in
                switch item {
                case let string as String:
                    continuation.resume(returning: string)
                case let data as Data:
                    let string = String(data: data, encoding: .utf8)
                    continuation.resume(returning: string)
                default:
                    continuation.resume(returning: nil)
                }
            }
        }
    }
}

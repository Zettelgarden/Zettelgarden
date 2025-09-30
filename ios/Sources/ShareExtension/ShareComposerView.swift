import SwiftUI
import UIKit
import ZettelgardenShared

struct ShareComposerView: View {
    @ObservedObject var viewModel: ShareExtensionViewModel
    let dismiss: () -> Void

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Title")) {
                    TextField("Title", text: $viewModel.shareItem.title)
                        .onChange(of: viewModel.shareItem.title) { _ in
                            viewModel.resetError()
                        }
                }

                Section(header: Text("Notes")) {
                    TextEditor(text: $viewModel.shareItem.body)
                        .frame(minHeight: 120)
                }

                if let url = viewModel.shareItem.url {
                    Section(header: Text("Link")) {
                        Text(url.absoluteString)
                            .font(.footnote)
                            .foregroundColor(.secondary)
                            .contextMenu {
                                Button(action: { UIPasteboard.general.string = url.absoluteString }) {
                                    Label("Copy", systemImage: "doc.on.doc")
                                }
                            }
                    }
                }

                Section(header: Text("Destination")) {
                    Picker("Destination", selection: $viewModel.destination) {
                        ForEach(DestinationType.allCases, id: \.self) { destination in
                            Text(destination.label).tag(destination)
                        }
                    }
                    .pickerStyle(.segmented)
                }

                if case .failed(let message) = viewModel.state {
                    Section {
                        Text(message)
                            .font(.footnote)
                            .foregroundColor(.red)
                    }
                }
            }
            .navigationTitle("Add to Zettelgarden")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel", action: dismiss)
                }
                ToolbarItem(placement: .confirmationAction) {
                    if viewModel.state == .submitting {
                        ProgressView()
                    } else {
                        Button("Save", action: submit)
                            .disabled(viewModel.shareItem.title.isEmpty)
                    }
                }
            }
        }
        .navigationViewStyle(.stack)
        .task(id: viewModel.state) {
            if viewModel.state == .completed {
                await MainActor.run(body: dismiss)
            }
        }
    }

    private func submit() {
        Task {
            await viewModel.submit()
        }
    }
}

#Preview {
    ShareComposerView(viewModel: ShareExtensionViewModel(initialItem: ShareItem(title: "Shared title", body: "", url: URL(string: "https://zettelgarden.app"))), dismiss: {})
}

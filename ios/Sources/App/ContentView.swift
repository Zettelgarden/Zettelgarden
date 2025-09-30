import SwiftUI
import ZettelgardenShared

struct ContentView: View {
    @EnvironmentObject private var appModel: AppModel

    @State private var baseURL: String = ""
    @State private var apiToken: String = ""
    @State private var defaultDestination: DestinationType = .card
    @State private var showSavedToast = false

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Backend")) {
                    TextField("API Base URL", text: $baseURL)
                        .keyboardType(.URL)
                        .autocapitalization(.none)
                    SecureField("API Token", text: $apiToken)
                }

                Section(header: Text("Defaults")) {
                    Picker("Default Destination", selection: $defaultDestination) {
                        ForEach(DestinationType.allCases, id: \.self) { destination in
                            Text(destination.label).tag(destination)
                        }
                    }
                    .pickerStyle(.segmented)
                }

                Section(footer: Text("The share extension uses these values via the shared app group.")) {
                    Button(action: saveConfiguration) {
                        Label("Save", systemImage: "checkmark.circle")
                    }
                }
            }
            .navigationTitle("Zettelgarden")
            .onAppear(perform: loadConfiguration)
            .toast(isPresented: $showSavedToast) {
                Label("Saved", systemImage: "checkmark")
                    .padding(.horizontal)
                    .padding(.vertical, 8)
                    .background(.thinMaterial, in: Capsule())
            }
        }
    }

    private func loadConfiguration() {
        let configuration = appModel.configuration
        baseURL = configuration.baseURL?.absoluteString ?? ""
        apiToken = configuration.apiToken
        defaultDestination = configuration.defaultDestination
    }

    private func saveConfiguration() {
        let url = URL(string: baseURL)
        let configuration = AppConfiguration(baseURL: url, apiToken: apiToken, defaultDestination: defaultDestination)
        appModel.save(configuration: configuration)
        withAnimation {
            showSavedToast = true
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) {
            withAnimation {
                showSavedToast = false
            }
        }
    }
}

private extension View {
    func toast<Content: View>(isPresented: Binding<Bool>, @ViewBuilder content: @escaping () -> Content) -> some View {
        overlay(alignment: .top) {
            if isPresented.wrappedValue {
                content()
                    .transition(.move(edge: .top).combined(with: .opacity))
                    .padding(.top, 12)
            }
        }
    }
}

#Preview {
    ContentView()
        .environmentObject(AppModel(store: InMemoryConfigurationStore(configuration: AppConfiguration(baseURL: URL(string: "https://example.com"), apiToken: "token", defaultDestination: .card))))
}

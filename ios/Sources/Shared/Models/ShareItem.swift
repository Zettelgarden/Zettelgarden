import Foundation

public struct ShareItem: Equatable {
    public var title: String
    public var body: String
    public var url: URL?

    public init(title: String = "", body: String = "", url: URL? = nil) {
        self.title = title
        self.body = body
        self.url = url
    }
}

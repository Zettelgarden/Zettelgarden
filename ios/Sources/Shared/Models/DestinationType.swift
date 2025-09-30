import Foundation

public enum DestinationType: String, CaseIterable, Codable {
    case card
    case task

    public var label: String {
        switch self {
        case .card:
            return "Card"
        case .task:
            return "Task"
        }
    }

    public var pathComponent: String {
        switch self {
        case .card:
            return "cards"
        case .task:
            return "tasks"
        }
    }
}

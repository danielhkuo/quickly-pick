// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import Foundation

// MARK: - Message Action

enum MessageAction: String {
    case vote
    case results
}

// MARK: - Message Payload

/// Encodes/decodes poll data for iMessage URL payloads
struct MessagePayload {
    let action: MessageAction
    let slug: String
    let title: String

    // MARK: - URL Encoding

    /// Encode payload to URL for iMessage
    func toURL() -> URL? {
        var components = URLComponents()
        components.scheme = "data"
        components.queryItems = [
            URLQueryItem(name: "action", value: action.rawValue),
            URLQueryItem(name: "slug", value: slug),
            URLQueryItem(name: "title", value: title)
        ]
        return components.url
    }

    /// Decode payload from URL
    static func fromURL(_ url: URL) -> MessagePayload? {
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              let queryItems = components.queryItems else {
            return nil
        }

        var actionString: String?
        var slug: String?
        var title: String?

        for item in queryItems {
            switch item.name {
            case "action":
                actionString = item.value
            case "slug":
                slug = item.value
            case "title":
                title = item.value
            default:
                break
            }
        }

        guard let actionStr = actionString,
              let action = MessageAction(rawValue: actionStr),
              let slugValue = slug,
              let titleValue = title else {
            return nil
        }

        return MessagePayload(action: action, slug: slugValue, title: titleValue)
    }

    // MARK: - Factory Methods

    /// Create a vote payload
    static func vote(slug: String, title: String) -> MessagePayload {
        MessagePayload(action: .vote, slug: slug, title: title)
    }

    /// Create a results payload
    static func results(slug: String, title: String) -> MessagePayload {
        MessagePayload(action: .results, slug: slug, title: title)
    }
}

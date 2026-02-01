// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import Foundation

// MARK: - Request Types

struct CreatePollRequest: Encodable {
    let title: String
    let description: String
    let creatorName: String

    enum CodingKeys: String, CodingKey {
        case title, description
        case creatorName = "creator_name"
    }
}

struct AddOptionRequest: Encodable {
    let label: String
}

struct ClaimUsernameRequest: Encodable {
    let username: String
}

struct SubmitBallotRequest: Encodable {
    let scores: [String: Double]
}

struct RegisterDeviceRequest: Encodable {
    let platform: String
}

// MARK: - Response Types

struct CreatePollResponse: Decodable {
    let pollId: String
    let adminKey: String

    enum CodingKeys: String, CodingKey {
        case pollId = "poll_id"
        case adminKey = "admin_key"
    }
}

struct AddOptionResponse: Decodable {
    let optionId: String

    enum CodingKeys: String, CodingKey {
        case optionId = "option_id"
    }
}

struct PublishPollResponse: Decodable {
    let shareSlug: String
    let shareUrl: String

    enum CodingKeys: String, CodingKey {
        case shareSlug = "share_slug"
        case shareUrl = "share_url"
    }
}

struct ClaimUsernameResponse: Decodable {
    let voterToken: String

    enum CodingKeys: String, CodingKey {
        case voterToken = "voter_token"
    }
}

struct SubmitBallotResponse: Decodable {
    let ballotId: String
    let message: String

    enum CodingKeys: String, CodingKey {
        case ballotId = "ballot_id"
        case message
    }
}

struct ClosePollResponse: Decodable {
    let closedAt: Date
    let snapshot: ResultSnapshot

    enum CodingKeys: String, CodingKey {
        case closedAt = "closed_at"
        case snapshot
    }
}

struct RegisterDeviceResponse: Decodable {
    let deviceId: String
    let isNew: Bool

    enum CodingKeys: String, CodingKey {
        case deviceId = "device_id"
        case isNew = "is_new"
    }
}

struct BallotCountResponse: Decodable {
    let ballotCount: Int

    enum CodingKeys: String, CodingKey {
        case ballotCount = "ballot_count"
    }
}

struct PollPreviewResponse: Decodable {
    let title: String
    let status: PollStatus
    let optionCount: Int
    let ballotCount: Int

    enum CodingKeys: String, CodingKey {
        case title, status
        case optionCount = "option_count"
        case ballotCount = "ballot_count"
    }
}

struct GetMyPollsResponse: Decodable {
    let polls: [DevicePollSummary]
}

struct GetMyBallotResponse: Decodable {
    let scores: [String: Double]
    let submittedAt: Date?
    let hasVoted: Bool

    enum CodingKeys: String, CodingKey {
        case scores
        case submittedAt = "submitted_at"
        case hasVoted = "has_voted"
    }
}

// MARK: - Error Response

struct APIErrorResponse: Decodable {
    let error: String
    let message: String?
}

// MARK: - API Error

enum APIError: LocalizedError {
    case networkError(Error)
    case httpError(statusCode: Int, message: String)
    case decodingError(Error)
    case invalidURL
    case noDeviceUUID

    var errorDescription: String? {
        switch self {
        case .networkError(let error):
            return "Network error: \(error.localizedDescription)"
        case .httpError(let code, let message):
            return "HTTP \(code): \(message)"
        case .decodingError(let error):
            return "Decoding error: \(error.localizedDescription)"
        case .invalidURL:
            return "Invalid URL"
        case .noDeviceUUID:
            return "Device not registered"
        }
    }
}

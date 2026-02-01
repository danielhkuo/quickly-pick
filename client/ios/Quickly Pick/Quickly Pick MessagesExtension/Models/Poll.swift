// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import Foundation

// MARK: - Poll Status

enum PollStatus: String, Codable {
    case draft
    case open
    case closed
}

// MARK: - Domain Models

struct Poll: Codable, Identifiable {
    let id: String
    let title: String
    let description: String
    let creatorName: String
    let method: String
    let status: PollStatus
    let shareSlug: String?
    let closesAt: Date?
    let closedAt: Date?
    let finalSnapshotId: String?
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id, title, description, method, status
        case creatorName = "creator_name"
        case shareSlug = "share_slug"
        case closesAt = "closes_at"
        case closedAt = "closed_at"
        case finalSnapshotId = "final_snapshot_id"
        case createdAt = "created_at"
    }
}

struct Option: Codable, Identifiable {
    let id: String
    let pollId: String
    let label: String

    enum CodingKeys: String, CodingKey {
        case id, label
        case pollId = "poll_id"
    }
}

struct PollWithOptions: Codable {
    let poll: Poll
    let options: [Option]
}

// MARK: - BMJ Result Types

struct OptionStats: Codable, Identifiable {
    let optionId: String
    let label: String
    let median: Double
    let p10: Double
    let p90: Double
    let mean: Double
    let negShare: Double
    let veto: Bool
    let rank: Int

    var id: String { optionId }

    enum CodingKeys: String, CodingKey {
        case label, median, p10, p90, mean, veto, rank
        case optionId = "option_id"
        case negShare = "neg_share"
    }
}

struct ResultSnapshot: Codable {
    let id: String
    let pollId: String
    let method: String
    let computedAt: Date
    let rankings: [OptionStats]
    let inputsHash: String

    enum CodingKeys: String, CodingKey {
        case id, method, rankings
        case pollId = "poll_id"
        case computedAt = "computed_at"
        case inputsHash = "inputs_hash"
    }
}

// MARK: - Device Poll Summary

struct DevicePollSummary: Codable, Identifiable {
    let pollId: String
    let title: String
    let status: PollStatus
    let shareSlug: String?
    let role: String
    let username: String?
    let ballotCount: Int
    let linkedAt: Date

    var id: String { pollId }

    enum CodingKeys: String, CodingKey {
        case title, status, role, username
        case pollId = "poll_id"
        case shareSlug = "share_slug"
        case ballotCount = "ballot_count"
        case linkedAt = "linked_at"
    }
}

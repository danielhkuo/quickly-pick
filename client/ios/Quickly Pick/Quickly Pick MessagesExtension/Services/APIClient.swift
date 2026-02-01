// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import Foundation

/// Network client for Quickly Pick API
final class APIClient {

    // MARK: - Configuration

    static let shared = APIClient()

    private let baseURL = URL(string: "https://quickly-pick-api.azurewebsites.net")!
    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder

    // MARK: - Initialization

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        self.session = URLSession(configuration: config)

        self.decoder = JSONDecoder()
        self.decoder.dateDecodingStrategy = .iso8601

        self.encoder = JSONEncoder()
        self.encoder.dateEncodingStrategy = .iso8601
    }

    // MARK: - Device Registration

    /// Register device with the backend
    func registerDevice() async throws -> RegisterDeviceResponse {
        let request = RegisterDeviceRequest(platform: "ios")
        return try await post("/devices/register", body: request)
    }

    // MARK: - Poll Management

    /// Create a new poll
    func createPoll(title: String, description: String = "", creatorName: String) async throws -> CreatePollResponse {
        let request = CreatePollRequest(title: title, description: description, creatorName: creatorName)
        return try await post("/polls", body: request)
    }

    /// Add an option to a poll
    func addOption(pollId: String, label: String, adminKey: String) async throws -> AddOptionResponse {
        let request = AddOptionRequest(label: label)
        return try await post("/polls/\(pollId)/options", body: request, headers: ["X-Admin-Key": adminKey])
    }

    /// Publish a poll to open voting
    func publishPoll(pollId: String, adminKey: String) async throws -> PublishPollResponse {
        return try await post("/polls/\(pollId)/publish", body: EmptyBody(), headers: ["X-Admin-Key": adminKey])
    }

    /// Close a poll
    func closePoll(pollId: String, adminKey: String) async throws -> ClosePollResponse {
        return try await post("/polls/\(pollId)/close", body: EmptyBody(), headers: ["X-Admin-Key": adminKey])
    }

    /// Get poll admin view
    func getPollAdmin(pollId: String, adminKey: String) async throws -> PollWithOptions {
        return try await get("/polls/\(pollId)/admin", headers: ["X-Admin-Key": adminKey])
    }

    // MARK: - Voting

    /// Get poll details by slug
    func getPoll(slug: String) async throws -> PollWithOptions {
        return try await get("/polls/\(slug)")
    }

    /// Claim a username for voting
    func claimUsername(slug: String, username: String) async throws -> ClaimUsernameResponse {
        let request = ClaimUsernameRequest(username: username)
        return try await post("/polls/\(slug)/claim-username", body: request)
    }

    /// Submit a ballot
    func submitBallot(slug: String, scores: [String: Double], voterToken: String) async throws -> SubmitBallotResponse {
        let request = SubmitBallotRequest(scores: scores)
        return try await post("/polls/\(slug)/ballots", body: request, headers: ["X-Voter-Token": voterToken])
    }

    // MARK: - Results

    /// Get poll results
    func getResults(slug: String) async throws -> ResultSnapshot {
        return try await get("/polls/\(slug)/results")
    }

    /// Get ballot count
    func getBallotCount(slug: String) async throws -> BallotCountResponse {
        return try await get("/polls/\(slug)/ballot-count")
    }

    /// Get poll preview
    func getPreview(slug: String) async throws -> PollPreviewResponse {
        return try await get("/polls/\(slug)/preview")
    }

    // MARK: - Device Polls

    /// Get polls for current device
    func getMyPolls() async throws -> [DevicePollSummary] {
        let response: GetMyPollsResponse = try await get("/devices/my-polls")
        return response.polls
    }

    // MARK: - Private Helpers

    private func get<T: Decodable>(_ path: String, headers: [String: String] = [:]) async throws -> T {
        var request = try makeRequest(path: path, method: "GET")
        for (key, value) in headers {
            request.setValue(value, forHTTPHeaderField: key)
        }
        return try await execute(request)
    }

    private func post<T: Decodable, B: Encodable>(_ path: String, body: B, headers: [String: String] = [:]) async throws -> T {
        var request = try makeRequest(path: path, method: "POST")
        request.httpBody = try encoder.encode(body)
        for (key, value) in headers {
            request.setValue(value, forHTTPHeaderField: key)
        }
        return try await execute(request)
    }

    private func makeRequest(path: String, method: String) throws -> URLRequest {
        guard let url = URL(string: path, relativeTo: baseURL) else {
            throw APIError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        // Add device UUID header
        let deviceUUID = KeychainService.shared.getOrCreateDeviceUUID()
        request.setValue(deviceUUID, forHTTPHeaderField: "X-Device-UUID")

        return request
    }

    private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
        let (data, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.networkError(URLError(.badServerResponse))
        }

        guard (200...299).contains(httpResponse.statusCode) else {
            let message: String
            if let errorResponse = try? decoder.decode(APIErrorResponse.self, from: data) {
                message = errorResponse.message ?? errorResponse.error
            } else if let text = String(data: data, encoding: .utf8), !text.isEmpty {
                message = text
            } else {
                message = "HTTP \(httpResponse.statusCode)"
            }
            throw APIError.httpError(statusCode: httpResponse.statusCode, message: message)
        }

        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw APIError.decodingError(error)
        }
    }
}

// MARK: - Empty Body for POST requests without body

private struct EmptyBody: Encodable {}

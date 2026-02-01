// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import Foundation
import SwiftUI

// MARK: - App State

enum AppState: Equatable {
    case compact
    case createPoll
    case claimUsername(slug: String, title: String)
    case vote(slug: String, title: String)
    case voteSubmitted(slug: String, title: String)
    case results(slug: String)
    case error(message: String)
}

// MARK: - Root View Model

@MainActor
final class RootViewModel: ObservableObject {

    // MARK: - Published Properties

    @Published var state: AppState = .compact
    @Published var isLoading = false

    // MARK: - Poll Creation State

    @Published var pollTitle = ""
    @Published var pollOptions: [String] = ["", ""]

    // MARK: - Claim Username State

    @Published var username = ""

    // MARK: - Voting State

    @Published var currentPoll: PollWithOptions?
    @Published var scores: [String: Double] = [:]

    // MARK: - Results State

    @Published var results: ResultSnapshot?

    // MARK: - Dependencies

    private let api = APIClient.shared
    private let keychain = KeychainService.shared

    // MARK: - Callbacks

    var onSendMessage: ((MessagePayload) -> Void)?
    var onRequestExpand: (() -> Void)?

    // MARK: - Navigation

    func showCreatePoll() {
        resetCreatePollState()
        state = .createPoll
        onRequestExpand?()
    }

    func showCompact() {
        state = .compact
    }

    /// Handle incoming message payload
    func handleIncomingMessage(_ payload: MessagePayload) {
        switch payload.action {
        case .vote:
            handleVoteAction(slug: payload.slug, title: payload.title)
        case .results:
            handleResultsAction(slug: payload.slug)
        }
        onRequestExpand?()
    }

    // MARK: - Poll Creation

    func addOption() {
        pollOptions.append("")
    }

    func removeOption(at index: Int) {
        guard pollOptions.count > 2 else { return }
        pollOptions.remove(at: index)
    }

    var canCreatePoll: Bool {
        !pollTitle.trimmingCharacters(in: .whitespaces).isEmpty &&
        pollOptions.filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }.count >= 2
    }

    func createAndSharePoll() {
        guard canCreatePoll else { return }

        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                // Create poll
                let createResponse = try await api.createPoll(
                    title: pollTitle.trimmingCharacters(in: .whitespaces),
                    creatorName: "iOS User"
                )

                let pollId = createResponse.pollId
                let adminKey = createResponse.adminKey

                // Store admin key
                keychain.setAdminKey(adminKey, forPollId: pollId)

                // Add options
                let validOptions = pollOptions.filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }
                for option in validOptions {
                    _ = try await api.addOption(
                        pollId: pollId,
                        label: option.trimmingCharacters(in: .whitespaces),
                        adminKey: adminKey
                    )
                }

                // Publish poll
                let publishResponse = try await api.publishPoll(pollId: pollId, adminKey: adminKey)

                // Send message
                let payload = MessagePayload.vote(slug: publishResponse.shareSlug, title: pollTitle)
                onSendMessage?(payload)

                // Reset and go back to compact
                resetCreatePollState()
                state = .compact

            } catch {
                state = .error(message: error.localizedDescription)
            }
        }
    }

    private func resetCreatePollState() {
        pollTitle = ""
        pollOptions = ["", ""]
    }

    // MARK: - Voting

    private func handleVoteAction(slug: String, title: String) {
        // Check if we already have a voter token
        if keychain.getVoterToken(forSlug: slug) != nil {
            loadPollForVoting(slug: slug, title: title)
        } else {
            state = .claimUsername(slug: slug, title: title)
        }
    }

    var canClaimUsername: Bool {
        !username.trimmingCharacters(in: .whitespaces).isEmpty
    }

    func claimUsernameAndVote(slug: String, title: String) {
        guard canClaimUsername else { return }

        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                let response = try await api.claimUsername(
                    slug: slug,
                    username: username.trimmingCharacters(in: .whitespaces)
                )

                keychain.setVoterToken(response.voterToken, forSlug: slug)
                username = ""

                loadPollForVoting(slug: slug, title: title)

            } catch let error as APIError {
                if case .httpError(let code, let message) = error, code == 409 {
                    state = .error(message: "Username '\(username)' is already taken. Please choose another.")
                } else {
                    state = .error(message: error.localizedDescription)
                }
            } catch {
                state = .error(message: error.localizedDescription)
            }
        }
    }

    private func loadPollForVoting(slug: String, title: String) {
        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                currentPoll = try await api.getPoll(slug: slug)

                // Initialize scores to 0.5 (meh) for all options
                scores = [:]
                if let poll = currentPoll {
                    for option in poll.options {
                        scores[option.id] = 0.5
                    }
                }

                state = .vote(slug: slug, title: title)

            } catch {
                state = .error(message: error.localizedDescription)
            }
        }
    }

    func submitBallot(slug: String, title: String) {
        guard let voterToken = keychain.getVoterToken(forSlug: slug) else {
            state = .error(message: "No voter token found")
            return
        }

        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                _ = try await api.submitBallot(slug: slug, scores: scores, voterToken: voterToken)
                state = .voteSubmitted(slug: slug, title: title)
            } catch {
                state = .error(message: error.localizedDescription)
            }
        }
    }

    // MARK: - Results

    private func handleResultsAction(slug: String) {
        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                results = try await api.getResults(slug: slug)
                state = .results(slug: slug)
            } catch {
                state = .error(message: error.localizedDescription)
            }
        }
    }

    func loadResults(slug: String) {
        handleResultsAction(slug: slug)
    }

    // MARK: - Error Handling

    func dismissError() {
        state = .compact
    }
}

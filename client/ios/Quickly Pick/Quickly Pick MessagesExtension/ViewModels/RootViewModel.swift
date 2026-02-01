// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import Foundation
import SwiftUI
internal import Combine

// MARK: - App State

enum AppState: Equatable {
    case compact
    case createPoll
    case myPolls
    case pollAdmin(pollId: String)
    case claimUsername(slug: String, title: String)
    case viewingBallot(slug: String, title: String)
    case vote(slug: String, title: String, isEditing: Bool)
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

    // MARK: - Existing Ballot State

    @Published var existingBallot: GetMyBallotResponse?

    // MARK: - My Polls State

    @Published var myPolls: [DevicePollSummary] = []

    // MARK: - Admin State

    @Published var adminPoll: PollWithOptions?
    @Published var adminBallotCount: Int = 0

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

    func showMyPolls() {
        loadMyPolls()
        onRequestExpand?()
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
        if let voterToken = keychain.getVoterToken(forSlug: slug) {
            // Check if user has already voted
            checkExistingBallot(slug: slug, title: title, voterToken: voterToken)
        } else {
            state = .claimUsername(slug: slug, title: title)
        }
    }

    private func checkExistingBallot(slug: String, title: String, voterToken: String) {
        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                let ballot = try await api.getMyBallot(slug: slug, voterToken: voterToken)
                existingBallot = ballot

                if ballot.hasVoted {
                    // User has already voted - show their ballot
                    currentPoll = try await api.getPoll(slug: slug)
                    state = .viewingBallot(slug: slug, title: title)
                } else {
                    // No existing vote - go to voting
                    loadPollForVoting(slug: slug, title: title, isEditing: false)
                }
            } catch {
                // If there's an error checking ballot, fall back to vote flow
                loadPollForVoting(slug: slug, title: title, isEditing: false)
            }
        }
    }

    /// Start editing an existing vote
    func editVote(slug: String, title: String) {
        guard let ballot = existingBallot else { return }

        // Pre-populate scores from existing ballot
        scores = ballot.scores
        state = .vote(slug: slug, title: title, isEditing: true)
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

                loadPollForVoting(slug: slug, title: title, isEditing: false)

            } catch let error as APIError {
                if case .httpError(let code, _) = error, code == 409 {
                    state = .error(message: "Username '\(username)' is already taken. Please choose another.")
                } else {
                    state = .error(message: error.localizedDescription)
                }
            } catch {
                state = .error(message: error.localizedDescription)
            }
        }
    }

    private func loadPollForVoting(slug: String, title: String, isEditing: Bool) {
        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                currentPoll = try await api.getPoll(slug: slug)

                // Initialize scores to 0.5 (meh) for all options (only if not editing)
                if !isEditing {
                    scores = [:]
                    if let poll = currentPoll {
                        for option in poll.options {
                            scores[option.id] = 0.5
                        }
                    }
                }

                state = .vote(slug: slug, title: title, isEditing: isEditing)

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

    // MARK: - My Polls

    func loadMyPolls() {
        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                myPolls = try await api.getMyPolls()
                state = .myPolls
            } catch {
                state = .error(message: error.localizedDescription)
            }
        }
    }

    /// Check if we are the admin of a given poll
    func isAdmin(forPollId pollId: String) -> Bool {
        keychain.getAdminKey(forPollId: pollId) != nil
    }

    /// Check if we are the admin of a poll by slug (uses currentPoll)
    func isAdminForCurrentPoll() -> Bool {
        guard let poll = currentPoll?.poll else { return false }
        return isAdmin(forPollId: poll.id)
    }

    /// Get the poll ID for the current poll (if available)
    func currentPollId() -> String? {
        currentPoll?.poll.id
    }

    /// Navigate to a poll from My Polls list
    func openPoll(_ poll: DevicePollSummary) {
        // If we're the admin, show admin view
        if isAdmin(forPollId: poll.pollId) {
            loadPollAdmin(pollId: poll.pollId)
        } else if let slug = poll.shareSlug {
            // If poll is closed, show results; otherwise show voting
            if poll.status == .closed {
                handleResultsAction(slug: slug)
            } else {
                handleVoteAction(slug: slug, title: poll.title)
            }
        }
        onRequestExpand?()
    }

    // MARK: - Admin Operations

    func loadPollAdmin(pollId: String) {
        guard let adminKey = keychain.getAdminKey(forPollId: pollId) else {
            state = .error(message: "Admin key not found")
            return
        }

        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                adminPoll = try await api.getPollAdmin(pollId: pollId, adminKey: adminKey)
                if let slug = adminPoll?.poll.shareSlug {
                    let countResponse = try await api.getBallotCount(slug: slug)
                    adminBallotCount = countResponse.ballotCount
                }
                state = .pollAdmin(pollId: pollId)
            } catch {
                state = .error(message: error.localizedDescription)
            }
        }
    }

    func closePoll(pollId: String) {
        guard let adminKey = keychain.getAdminKey(forPollId: pollId) else {
            state = .error(message: "Admin key not found")
            return
        }

        Task {
            isLoading = true
            defer { isLoading = false }

            do {
                let response = try await api.closePoll(pollId: pollId, adminKey: adminKey)
                results = response.snapshot

                // Navigate to results if we have a slug (check adminPoll first, then currentPoll)
                if let slug = adminPoll?.poll.shareSlug ?? currentPoll?.poll.shareSlug {
                    state = .results(slug: slug)
                } else {
                    state = .compact
                }
            } catch {
                state = .error(message: error.localizedDescription)
            }
        }
    }

    func shareResults(slug: String, title: String) {
        let payload = MessagePayload.results(slug: slug, title: title)
        onSendMessage?(payload)
    }

    // MARK: - Error Handling

    func dismissError() {
        state = .compact
    }
}

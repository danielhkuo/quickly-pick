// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import SwiftUI

struct ViewingBallotView: View {
    @ObservedObject var viewModel: RootViewModel
    let slug: String
    let title: String

    @State private var showCloseConfirmation = false

    private var isAdmin: Bool {
        viewModel.isAdminForCurrentPoll()
    }

    private var pollId: String? {
        viewModel.currentPollId()
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    headerSection

                    if let poll = viewModel.currentPoll {
                        scoresSection(poll: poll)
                    }

                    actionsSection
                }
                .padding()
            }
            .navigationTitle("Your Vote")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") {
                        viewModel.showCompact()
                    }
                }
            }
            .confirmationDialog(
                "Close Poll?",
                isPresented: $showCloseConfirmation,
                titleVisibility: .visible
            ) {
                Button("Close Poll", role: .destructive) {
                    if let pollId = pollId {
                        viewModel.closePoll(pollId: pollId)
                    }
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("This will end voting and calculate final results. This action cannot be undone.")
            }
        }
    }

    private var headerSection: some View {
        VStack(spacing: 12) {
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 48))
                .foregroundColor(.green)

            Text(title)
                .font(.title3.bold())
                .multilineTextAlignment(.center)

            if let ballot = viewModel.existingBallot, let submittedAt = ballot.submittedAt {
                Text("Submitted \(submittedAt.formatted(date: .abbreviated, time: .shortened))")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
        }
    }

    @ViewBuilder
    private func scoresSection(poll: PollWithOptions) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Your Ratings")
                .font(.headline)

            ForEach(poll.options) { option in
                scoreRow(for: option)
            }
        }
    }

    private func scoreRow(for option: Option) -> some View {
        let score = viewModel.existingBallot?.scores[option.id] ?? 0.5

        return HStack {
            Text(option.label)
                .font(.body)

            Spacer()

            scoreLabel(for: score)
        }
        .padding(.vertical, 8)
        .padding(.horizontal, 12)
        .background(Color(.systemGray6))
        .cornerRadius(8)
    }

    @ViewBuilder
    private func scoreLabel(for score: Double) -> some View {
        let (text, color) = scoreLabelInfo(for: score)

        Text(text)
            .font(.subheadline.weight(.medium))
            .foregroundColor(color)
    }

    private func scoreLabelInfo(for score: Double) -> (String, Color) {
        if score >= 0.8 {
            return ("Love", .green)
        } else if score >= 0.6 {
            return ("Like", Color(red: 0.4, green: 0.7, blue: 0.4))
        } else if score >= 0.4 {
            return ("Meh", .gray)
        } else if score >= 0.2 {
            return ("Dislike", .orange)
        } else {
            return ("Hate", .red)
        }
    }

    private var actionsSection: some View {
        VStack(spacing: 12) {
            // Edit Vote button
            Button(action: { viewModel.editVote(slug: slug, title: title) }) {
                Label("Edit Vote", systemImage: "pencil")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.bordered)
            .controlSize(.large)

            // Admin: Close Poll button
            if isAdmin {
                Button(action: { showCloseConfirmation = true }) {
                    Label("Close Poll & Show Results", systemImage: "lock.fill")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .tint(.red)
                .controlSize(.large)
            }
        }
    }
}

#Preview("Voter") {
    let viewModel = RootViewModel()
    viewModel.currentPoll = PollWithOptions(
        poll: Poll(
            id: "poll1",
            title: "Where should we eat?",
            description: "",
            creatorName: "Test User",
            method: "bmj",
            status: .open,
            shareSlug: "abc123",
            closesAt: nil,
            closedAt: nil,
            finalSnapshotId: nil,
            createdAt: Date()
        ),
        options: [
            Option(id: "opt1", pollId: "poll1", label: "Pizza"),
            Option(id: "opt2", pollId: "poll1", label: "Tacos"),
            Option(id: "opt3", pollId: "poll1", label: "Sushi")
        ]
    )
    viewModel.existingBallot = GetMyBallotResponse(
        scores: ["opt1": 0.9, "opt2": 0.3, "opt3": 0.5],
        submittedAt: Date().addingTimeInterval(-3600),
        hasVoted: true
    )

    return ViewingBallotView(viewModel: viewModel, slug: "abc123", title: "Where should we eat?")
}

#Preview("Admin") {
    let viewModel = RootViewModel()
    viewModel.currentPoll = PollWithOptions(
        poll: Poll(
            id: "poll1",
            title: "Where should we eat?",
            description: "",
            creatorName: "iOS User",
            method: "bmj",
            status: .open,
            shareSlug: "abc123",
            closesAt: nil,
            closedAt: nil,
            finalSnapshotId: nil,
            createdAt: Date()
        ),
        options: [
            Option(id: "opt1", pollId: "poll1", label: "Pizza"),
            Option(id: "opt2", pollId: "poll1", label: "Tacos"),
            Option(id: "opt3", pollId: "poll1", label: "Sushi")
        ]
    )
    viewModel.existingBallot = GetMyBallotResponse(
        scores: ["opt1": 0.85, "opt2": 0.15, "opt3": 0.5],
        submittedAt: Date().addingTimeInterval(-7200),
        hasVoted: true
    )
    // Note: In preview, we can't actually set admin key in keychain
    // The "Close Poll" button won't show without it

    return ViewingBallotView(viewModel: viewModel, slug: "abc123", title: "Where should we eat?")
}

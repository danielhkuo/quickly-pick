// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import SwiftUI

struct RootView: View {
    @ObservedObject var viewModel: RootViewModel

    var body: some View {
        ZStack {
            content
                .disabled(viewModel.isLoading)

            if viewModel.isLoading {
                loadingOverlay
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        switch viewModel.state {
        case .compact:
            CompactView(viewModel: viewModel)

        case .createPoll:
            CreatePollView(viewModel: viewModel)

        case .myPolls:
            MyPollsView(viewModel: viewModel)

        case .pollAdmin(let pollId):
            PollAdminView(viewModel: viewModel, pollId: pollId)

        case .claimUsername(let slug, let title):
            ClaimUsernameView(viewModel: viewModel, slug: slug, title: title)

        case .viewingBallot(let slug, let title):
            ViewingBallotView(viewModel: viewModel, slug: slug, title: title)

        case .vote(let slug, let title, let isEditing):
            VoteView(viewModel: viewModel, slug: slug, title: title, isEditing: isEditing)

        case .voteSubmitted(let slug, let title):
            VoteSubmittedView(viewModel: viewModel, slug: slug, title: title)

        case .results(let slug):
            ResultsView(viewModel: viewModel, slug: slug)

        case .error(let message):
            ErrorView(message: message, onDismiss: viewModel.dismissError)
        }
    }

    private var loadingOverlay: some View {
        ZStack {
            Color.black.opacity(0.3)
                .ignoresSafeArea()

            ProgressView()
                .scaleEffect(1.5)
                .tint(.white)
        }
    }
}

// MARK: - Vote Submitted View

struct VoteSubmittedView: View {
    @ObservedObject var viewModel: RootViewModel
    let slug: String
    let title: String

    var body: some View {
        VStack(spacing: 24) {
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 64))
                .foregroundColor(.green)

            Text("Vote Submitted!")
                .font(.title2.bold())

            Text("Your vote for \"\(title)\" has been recorded.")
                .font(.body)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            Button(action: { viewModel.loadResults(slug: slug) }) {
                Label("View Results", systemImage: "chart.bar.fill")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.bordered)
            .controlSize(.large)

            Button("Done") {
                viewModel.showCompact()
            }
            .foregroundColor(.secondary)
        }
        .padding()
    }
}

// MARK: - Error View

struct ErrorView: View {
    let message: String
    let onDismiss: () -> Void

    var body: some View {
        VStack(spacing: 24) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 64))
                .foregroundColor(.orange)

            Text("Something went wrong")
                .font(.title2.bold())

            Text(message)
                .font(.body)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            Button("Try Again", action: onDismiss)
                .buttonStyle(.borderedProminent)
                .controlSize(.large)
        }
        .padding()
    }
}

#Preview("Compact") {
    RootView(viewModel: RootViewModel())
}

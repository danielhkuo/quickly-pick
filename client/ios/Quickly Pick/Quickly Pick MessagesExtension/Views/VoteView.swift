// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import SwiftUI

struct VoteView: View {
    @ObservedObject var viewModel: RootViewModel
    let slug: String
    let title: String

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    headerSection

                    if let poll = viewModel.currentPoll {
                        optionsSection(poll: poll)
                    }

                    submitButton
                }
                .padding()
            }
            .navigationTitle("Vote")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        viewModel.showCompact()
                    }
                }
            }
        }
    }

    private var headerSection: some View {
        VStack(spacing: 8) {
            Text(title)
                .font(.title3.bold())
                .multilineTextAlignment(.center)

            Text("Drag each slider to express how you feel")
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
    }

    @ViewBuilder
    private func optionsSection(poll: PollWithOptions) -> some View {
        VStack(spacing: 16) {
            ForEach(poll.options) { option in
                BipolarSlider(
                    label: option.label,
                    value: Binding(
                        get: { viewModel.scores[option.id] ?? 0.5 },
                        set: { viewModel.scores[option.id] = $0 }
                    )
                )
                .padding()
                .background(Color(.systemBackground))
                .cornerRadius(12)
                .shadow(color: .black.opacity(0.05), radius: 2, y: 1)
            }
        }
    }

    private var submitButton: some View {
        Button(action: { viewModel.submitBallot(slug: slug, title: title) }) {
            Label("Submit Vote", systemImage: "checkmark.circle.fill")
                .frame(maxWidth: .infinity)
        }
        .buttonStyle(.borderedProminent)
        .controlSize(.large)
        .padding(.top, 8)
    }
}

#Preview {
    let viewModel = RootViewModel()
    viewModel.currentPoll = PollWithOptions(
        poll: Poll(
            id: "1",
            title: "Where to eat?",
            description: "",
            creatorName: "Test",
            method: "bmj",
            status: .open,
            shareSlug: "abc123",
            closesAt: nil,
            closedAt: nil,
            finalSnapshotId: nil,
            createdAt: Date()
        ),
        options: [
            Option(id: "opt1", pollId: "1", label: "Pizza"),
            Option(id: "opt2", pollId: "1", label: "Tacos"),
            Option(id: "opt3", pollId: "1", label: "Sushi")
        ]
    )
    viewModel.scores = ["opt1": 0.5, "opt2": 0.5, "opt3": 0.5]

    return VoteView(viewModel: viewModel, slug: "abc123", title: "Where to eat?")
}

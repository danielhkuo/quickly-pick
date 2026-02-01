// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import SwiftUI

struct ResultsView: View {
    @ObservedObject var viewModel: RootViewModel
    let slug: String

    var body: some View {
        NavigationStack {
            ScrollView {
                if let results = viewModel.results {
                    resultsContent(results: results)
                } else {
                    ProgressView("Loading results...")
                        .padding(.top, 100)
                }
            }
            .navigationTitle("Results")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") {
                        viewModel.showCompact()
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func resultsContent(results: ResultSnapshot) -> some View {
        VStack(spacing: 20) {
            headerSection(results: results)

            rankingsSection(results: results)
        }
        .padding()
    }

    private func headerSection(results: ResultSnapshot) -> some View {
        VStack(spacing: 4) {
            Image(systemName: "trophy.fill")
                .font(.system(size: 48))
                .foregroundColor(.yellow)

            Text("Poll Results")
                .font(.title2.bold())

            Text("Ranked by Balanced Majority Judgment")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.bottom, 8)
    }

    private func rankingsSection(results: ResultSnapshot) -> some View {
        VStack(spacing: 12) {
            ForEach(results.rankings) { option in
                ResultRow(option: option, isWinner: option.rank == 1)
            }
        }
    }
}

// MARK: - Result Row

struct ResultRow: View {
    let option: OptionStats
    let isWinner: Bool

    private var sentimentColor: Color {
        switch option.median {
        case 0...0.3:
            return .red
        case 0.3...0.7:
            return .gray
        default:
            return .green
        }
    }

    private var sentimentLabel: String {
        switch option.median {
        case 0...0.1:
            return "Strongly disliked"
        case 0.1...0.3:
            return "Disliked"
        case 0.3...0.7:
            return "Neutral"
        case 0.7...0.9:
            return "Liked"
        default:
            return "Strongly liked"
        }
    }

    var body: some View {
        HStack(spacing: 12) {
            // Rank badge
            rankBadge

            // Option info
            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(option.label)
                        .font(.headline)

                    if option.veto {
                        vetoIndicator
                    }
                }

                HStack(spacing: 8) {
                    Text(sentimentLabel)
                        .font(.subheadline)
                        .foregroundColor(sentimentColor)

                    Text("•")
                        .foregroundColor(.secondary)

                    Text("Median: \(Int(option.median * 100))%")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            Spacer()

            // Score bar
            scoreBar
        }
        .padding()
        .background(isWinner ? Color.yellow.opacity(0.1) : Color(.systemBackground))
        .cornerRadius(12)
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(isWinner ? Color.yellow : Color.clear, lineWidth: 2)
        )
        .shadow(color: .black.opacity(0.05), radius: 2, y: 1)
    }

    private var rankBadge: some View {
        ZStack {
            Circle()
                .fill(isWinner ? Color.yellow : Color(.systemGray5))
                .frame(width: 36, height: 36)

            if isWinner {
                Image(systemName: "crown.fill")
                    .font(.system(size: 16))
                    .foregroundColor(.orange)
            } else {
                Text("#\(option.rank)")
                    .font(.caption.bold())
                    .foregroundColor(.secondary)
            }
        }
    }

    private var vetoIndicator: some View {
        HStack(spacing: 2) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.caption2)
            Text("Vetoed")
                .font(.caption2)
        }
        .foregroundColor(.red)
        .padding(.horizontal, 6)
        .padding(.vertical, 2)
        .background(Color.red.opacity(0.1))
        .cornerRadius(4)
    }

    private var scoreBar: some View {
        GeometryReader { geometry in
            ZStack(alignment: .leading) {
                // Background
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color(.systemGray5))

                // Fill
                RoundedRectangle(cornerRadius: 4)
                    .fill(sentimentColor.opacity(0.8))
                    .frame(width: geometry.size.width * option.median)
            }
        }
        .frame(width: 60, height: 8)
    }
}

#Preview {
    let viewModel = RootViewModel()
    viewModel.results = ResultSnapshot(
        id: "snap1",
        pollId: "poll1",
        method: "bmj",
        computedAt: Date(),
        rankings: [
            OptionStats(optionId: "1", label: "Sushi", median: 0.85, p10: 0.6, p90: 0.95, mean: 0.82, negShare: 0.05, veto: false, rank: 1),
            OptionStats(optionId: "2", label: "Pizza", median: 0.65, p10: 0.4, p90: 0.8, mean: 0.62, negShare: 0.15, veto: false, rank: 2),
            OptionStats(optionId: "3", label: "Fast Food", median: 0.25, p10: 0.1, p90: 0.5, mean: 0.28, negShare: 0.55, veto: true, rank: 3)
        ],
        inputsHash: "abc123"
    )

    return ResultsView(viewModel: viewModel, slug: "abc123")
}

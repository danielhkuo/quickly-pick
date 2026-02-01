// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import SwiftUI

struct ClaimUsernameView: View {
    @ObservedObject var viewModel: RootViewModel
    let slug: String
    let title: String
    @FocusState private var isUsernameFocused: Bool

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Spacer()

                Image(systemName: "person.crop.circle.badge.plus")
                    .font(.system(size: 64))
                    .foregroundColor(.accentColor)

                VStack(spacing: 8) {
                    Text("Join the Poll")
                        .font(.title2.bold())

                    Text("Choose a display name to vote on")
                        .font(.body)
                        .foregroundColor(.secondary)

                    Text("\"\(title)\"")
                        .font(.headline)
                        .foregroundColor(.primary)
                }

                TextField("Your name", text: $viewModel.username)
                    .textFieldStyle(.roundedBorder)
                    .textContentType(.nickname)
                    .autocorrectionDisabled()
                    .textInputAutocapitalization(.words)
                    .focused($isUsernameFocused)
                    .padding(.horizontal, 40)
                    .onSubmit {
                        if viewModel.canClaimUsername {
                            viewModel.claimUsernameAndVote(slug: slug, title: title)
                        }
                    }

                Button(action: { viewModel.claimUsernameAndVote(slug: slug, title: title) }) {
                    Text("Continue")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.large)
                .disabled(!viewModel.canClaimUsername)
                .padding(.horizontal, 40)

                Spacer()
                Spacer()
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
            .onAppear {
                isUsernameFocused = true
            }
        }
    }
}

#Preview {
    ClaimUsernameView(
        viewModel: RootViewModel(),
        slug: "abc123",
        title: "Where should we eat?"
    )
}

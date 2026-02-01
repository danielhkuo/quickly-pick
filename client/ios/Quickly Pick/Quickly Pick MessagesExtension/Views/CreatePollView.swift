// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import SwiftUI

struct CreatePollView: View {
    @ObservedObject var viewModel: RootViewModel
    @FocusState private var focusedField: Field?

    private enum Field: Hashable {
        case title
        case option(Int)
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("What should we decide?", text: $viewModel.pollTitle)
                        .focused($focusedField, equals: .title)
                } header: {
                    Text("Question")
                }

                Section {
                    ForEach(viewModel.pollOptions.indices, id: \.self) { index in
                        HStack {
                            TextField("Option \(index + 1)", text: $viewModel.pollOptions[index])
                                .focused($focusedField, equals: .option(index))

                            if viewModel.pollOptions.count > 2 {
                                Button(action: { viewModel.removeOption(at: index) }) {
                                    Image(systemName: "minus.circle.fill")
                                        .foregroundColor(.red)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }

                    Button(action: viewModel.addOption) {
                        Label("Add Option", systemImage: "plus.circle")
                    }
                } header: {
                    Text("Options")
                } footer: {
                    Text("Add at least 2 options for people to vote on")
                }
            }
            .navigationTitle("New Poll")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        viewModel.showCompact()
                    }
                }

                ToolbarItem(placement: .confirmationAction) {
                    Button("Create & Share") {
                        viewModel.createAndSharePoll()
                    }
                    .fontWeight(.semibold)
                    .disabled(!viewModel.canCreatePoll)
                }
            }
            .onAppear {
                focusedField = .title
            }
        }
    }
}

#Preview {
    CreatePollView(viewModel: RootViewModel())
}

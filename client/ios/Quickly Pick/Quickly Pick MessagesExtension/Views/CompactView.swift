// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import SwiftUI

struct CompactView: View {
    @ObservedObject var viewModel: RootViewModel

    var body: some View {
        HStack {
            Spacer()

            Button(action: viewModel.showCreatePoll) {
                Label("New Poll", systemImage: "plus.circle.fill")
                    .font(.headline)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)

            Spacer()
        }
        .padding()
    }
}

#Preview {
    CompactView(viewModel: RootViewModel())
}

// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import SwiftUI

struct BipolarSlider: View {
    let label: String
    @Binding var value: Double

    private var semanticLabel: String {
        switch value {
        case 0...0.1:
            return "Strongly dislike"
        case 0.1...0.3:
            return "Dislike"
        case 0.3...0.7:
            return "Neutral"
        case 0.7...0.9:
            return "Like"
        default:
            return "Strongly like"
        }
    }

    private var sliderColor: Color {
        switch value {
        case 0...0.3:
            return .red
        case 0.3...0.7:
            return .gray
        default:
            return .green
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(label)
                    .font(.headline)

                Spacer()

                Text(semanticLabel)
                    .font(.subheadline)
                    .foregroundColor(sliderColor)
                    .fontWeight(.medium)
            }

            Slider(value: $value, in: 0...1, step: 0.01)
                .tint(sliderColor)

            HStack {
                Text("Hate")
                    .font(.caption)
                    .foregroundColor(.red)

                Spacer()

                Text("Meh")
                    .font(.caption)
                    .foregroundColor(.gray)

                Spacer()

                Text("Love")
                    .font(.caption)
                    .foregroundColor(.green)
            }
        }
        .padding(.vertical, 8)
    }
}

#Preview {
    struct PreviewWrapper: View {
        @State private var value = 0.5

        var body: some View {
            VStack {
                BipolarSlider(label: "Pizza", value: $value)
                BipolarSlider(label: "Tacos", value: .constant(0.1))
                BipolarSlider(label: "Sushi", value: .constant(0.9))
            }
            .padding()
        }
    }

    return PreviewWrapper()
}

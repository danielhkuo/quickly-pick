// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import UIKit
import Messages
import SwiftUI

class MessagesViewController: MSMessagesAppViewController {

    // MARK: - Properties

    private var rootViewModel: RootViewModel!
    private var hostingController: UIHostingController<RootView>!

    // MARK: - Lifecycle

    override func viewDidLoad() {
        super.viewDidLoad()

        setupViewModel()
        setupSwiftUIView()
    }

    private func setupViewModel() {
        rootViewModel = RootViewModel()

        rootViewModel.onSendMessage = { [weak self] payload in
            self?.sendPollMessage(payload: payload)
        }

        rootViewModel.onRequestExpand = { [weak self] in
            self?.requestPresentationStyle(.expanded)
        }
    }

    private func setupSwiftUIView() {
        let rootView = RootView(viewModel: rootViewModel)
        hostingController = UIHostingController(rootView: rootView)

        addChild(hostingController)
        view.addSubview(hostingController.view)
        hostingController.view.translatesAutoresizingMaskIntoConstraints = false

        NSLayoutConstraint.activate([
            hostingController.view.topAnchor.constraint(equalTo: view.topAnchor),
            hostingController.view.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            hostingController.view.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            hostingController.view.trailingAnchor.constraint(equalTo: view.trailingAnchor)
        ])

        hostingController.didMove(toParent: self)
    }

    // MARK: - Conversation Handling

    override func willBecomeActive(with conversation: MSConversation) {
        updateForPresentationStyle(presentationStyle)

        // Check if opened from a message
        if let message = conversation.selectedMessage,
           let url = message.url,
           let payload = MessagePayload.fromURL(url) {
            rootViewModel.handleIncomingMessage(payload)
        }
    }

    override func didResignActive(with conversation: MSConversation) {
        // Reset to compact state when dismissed
    }

    override func didReceive(_ message: MSMessage, conversation: MSConversation) {
        // Handle incoming message while active
        if let url = message.url,
           let payload = MessagePayload.fromURL(url) {
            rootViewModel.handleIncomingMessage(payload)
        }
    }

    override func didStartSending(_ message: MSMessage, conversation: MSConversation) {
        // Message sent successfully
    }

    override func didCancelSending(_ message: MSMessage, conversation: MSConversation) {
        // User cancelled - could reset state if needed
    }

    // MARK: - Presentation Style

    override func willTransition(to presentationStyle: MSMessagesAppPresentationStyle) {
        updateForPresentationStyle(presentationStyle)
    }

    override func didTransition(to presentationStyle: MSMessagesAppPresentationStyle) {
        // Finalize any behaviors after transition
    }

    private func updateForPresentationStyle(_ style: MSMessagesAppPresentationStyle) {
        switch style {
        case .compact:
            rootViewModel.showCompact()
        case .expanded, .transcript:
            // Keep current state when expanded
            break
        @unknown default:
            break
        }
    }

    // MARK: - Message Sending

    private func sendPollMessage(payload: MessagePayload) {
        guard let conversation = activeConversation else { return }

        let message = MSMessage(session: conversation.selectedMessage?.session ?? MSSession())
        message.url = payload.toURL()

        let layout = MSMessageTemplateLayout()
        layout.caption = payload.title

        switch payload.action {
        case .vote:
            layout.subcaption = "Tap to vote!"
            layout.image = createPollImage(title: payload.title)
        case .results:
            layout.subcaption = "Tap to view results"
            layout.image = createResultsImage(title: payload.title)
        }

        message.layout = layout
        message.summaryText = "Poll: \(payload.title)"

        conversation.insert(message) { error in
            if let error = error {
                print("Failed to insert message: \(error)")
            }
        }

        dismiss()
    }

    // MARK: - Image Generation

    private func createPollImage(title: String) -> UIImage {
        let size = CGSize(width: 300, height: 200)
        let renderer = UIGraphicsImageRenderer(size: size)

        return renderer.image { context in
            // Background
            UIColor.systemBlue.setFill()
            context.fill(CGRect(origin: .zero, size: size))

            // Icon
            let iconConfig = UIImage.SymbolConfiguration(pointSize: 48, weight: .medium)
            if let icon = UIImage(systemName: "chart.bar.doc.horizontal", withConfiguration: iconConfig) {
                icon.withTintColor(.white, renderingMode: .alwaysOriginal)
                    .draw(at: CGPoint(x: (size.width - icon.size.width) / 2, y: 40))
            }

            // Text
            let paragraphStyle = NSMutableParagraphStyle()
            paragraphStyle.alignment = .center

            let attributes: [NSAttributedString.Key: Any] = [
                .font: UIFont.systemFont(ofSize: 18, weight: .semibold),
                .foregroundColor: UIColor.white,
                .paragraphStyle: paragraphStyle
            ]

            let textRect = CGRect(x: 16, y: 120, width: size.width - 32, height: 60)
            title.draw(in: textRect, withAttributes: attributes)
        }
    }

    private func createResultsImage(title: String) -> UIImage {
        let size = CGSize(width: 300, height: 200)
        let renderer = UIGraphicsImageRenderer(size: size)

        return renderer.image { context in
            // Background
            UIColor.systemGreen.setFill()
            context.fill(CGRect(origin: .zero, size: size))

            // Icon
            let iconConfig = UIImage.SymbolConfiguration(pointSize: 48, weight: .medium)
            if let icon = UIImage(systemName: "trophy.fill", withConfiguration: iconConfig) {
                icon.withTintColor(.white, renderingMode: .alwaysOriginal)
                    .draw(at: CGPoint(x: (size.width - icon.size.width) / 2, y: 40))
            }

            // Text
            let paragraphStyle = NSMutableParagraphStyle()
            paragraphStyle.alignment = .center

            let attributes: [NSAttributedString.Key: Any] = [
                .font: UIFont.systemFont(ofSize: 18, weight: .semibold),
                .foregroundColor: UIColor.white,
                .paragraphStyle: paragraphStyle
            ]

            let textRect = CGRect(x: 16, y: 120, width: size.width - 32, height: 60)
            "Results: \(title)".draw(in: textRect, withAttributes: attributes)
        }
    }
}

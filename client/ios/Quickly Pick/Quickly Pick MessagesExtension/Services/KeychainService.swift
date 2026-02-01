// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

import Foundation
import Security

/// Manages secure storage of device identity and authentication tokens
final class KeychainService {

    // MARK: - Keychain Keys

    private enum Keys {
        static let deviceUUID = "com.quicklypick.device-uuid"
        static let adminKeyPrefix = "com.quicklypick.admin-key."
        static let voterTokenPrefix = "com.quicklypick.voter-token."
    }

    // MARK: - Shared Instance

    static let shared = KeychainService()
    private init() {}

    // MARK: - Device UUID

    /// Get or create the device UUID
    func getOrCreateDeviceUUID() -> String {
        if let existing = getString(forKey: Keys.deviceUUID) {
            return existing
        }

        let newUUID = UUID().uuidString
        setString(newUUID, forKey: Keys.deviceUUID)
        return newUUID
    }

    /// Get device UUID if it exists
    var deviceUUID: String? {
        getString(forKey: Keys.deviceUUID)
    }

    // MARK: - Admin Keys

    /// Store admin key for a poll
    func setAdminKey(_ adminKey: String, forPollId pollId: String) {
        setString(adminKey, forKey: Keys.adminKeyPrefix + pollId)
    }

    /// Get admin key for a poll
    func getAdminKey(forPollId pollId: String) -> String? {
        getString(forKey: Keys.adminKeyPrefix + pollId)
    }

    /// Remove admin key for a poll
    func removeAdminKey(forPollId pollId: String) {
        removeItem(forKey: Keys.adminKeyPrefix + pollId)
    }

    // MARK: - Voter Tokens

    /// Store voter token for a poll (keyed by slug)
    func setVoterToken(_ voterToken: String, forSlug slug: String) {
        setString(voterToken, forKey: Keys.voterTokenPrefix + slug)
    }

    /// Get voter token for a poll (keyed by slug)
    func getVoterToken(forSlug slug: String) -> String? {
        getString(forKey: Keys.voterTokenPrefix + slug)
    }

    /// Remove voter token for a poll
    func removeVoterToken(forSlug slug: String) {
        removeItem(forKey: Keys.voterTokenPrefix + slug)
    }

    // MARK: - Generic Keychain Operations

    private func setString(_ value: String, forKey key: String) {
        guard let data = value.data(using: .utf8) else { return }

        // Delete existing item first
        removeItem(forKey: key)

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock
        ]

        SecItemAdd(query as CFDictionary, nil)
    }

    private func getString(forKey key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess,
              let data = result as? Data,
              let string = String(data: data, encoding: .utf8) else {
            return nil
        }

        return string
    }

    private func removeItem(forKey key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key
        ]

        SecItemDelete(query as CFDictionary)
    }
}

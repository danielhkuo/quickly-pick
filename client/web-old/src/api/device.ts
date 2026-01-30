const DEVICE_UUID_KEY = 'device_uuid'

/**
 * Get or create a persistent device UUID.
 * Uses crypto.randomUUID() for UUID v4 generation.
 * Persisted in localStorage for device identity across sessions.
 */
export function getDeviceUUID(): string {
  let uuid = localStorage.getItem(DEVICE_UUID_KEY)

  if (!uuid) {
    uuid = crypto.randomUUID()
    localStorage.setItem(DEVICE_UUID_KEY, uuid)
  }

  return uuid
}

/**
 * Clear the device UUID (useful for testing or reset scenarios)
 */
export function clearDeviceUUID(): void {
  localStorage.removeItem(DEVICE_UUID_KEY)
}

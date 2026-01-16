export const AUTH_CHANGED_EVENT = 'image-code:auth-changed'

export function emitAuthChanged(): void {
  try {
    window.dispatchEvent(new CustomEvent(AUTH_CHANGED_EVENT))
  } catch {
    // ignore (SSR / weird environments)
  }
}

import {
  getAuthSession,
  updateAccount,
  login as loginRequest,
  logout as logoutRequest,
  setCsrfToken,
  setupOwner,
  type AuthSession,
} from "./api";
import { signInWithPasskey } from "./passkey";

export type AuthStatus = "loading" | "setup" | "login" | "ready" | "error";

// App-wide owner session state. The session cookie is HttpOnly, so the app
// learns who is logged in only by asking the server.
export const auth = $state<{
  status: AuthStatus;
  username: string;
  displayName: string;
  passkeysEnabled: boolean;
}>({
  status: "loading",
  username: "",
  displayName: "",
  passkeysEnabled: false,
});

function apply(session: AuthSession): void {
  setCsrfToken(session.csrf_token ?? "");
  auth.username = session.username ?? "";
  auth.displayName = session.display_name ?? "";
  auth.passkeysEnabled = session.passkeys_enabled ?? auth.passkeysEnabled;
  auth.status = session.authenticated
    ? "ready"
    : session.setup_required
      ? "setup"
      : "login";
}

export async function loadSession(): Promise<void> {
  try {
    apply(await getAuthSession());
  } catch {
    auth.status = "error";
  }
}

export async function login(username: string, password: string) {
  apply(await loginRequest(username, password));
}

export async function passkeyLogin() {
  apply(await signInWithPasskey());
}

export async function setup(username: string, password: string) {
  apply(await setupOwner(username, password));
}

export async function logout(): Promise<void> {
  try {
    await logoutRequest();
  } finally {
    setCsrfToken("");
    auth.username = "";
    auth.status = "login";
  }
}

export function sessionExpired(): void {
  setCsrfToken("");
  if (auth.status === "ready") auth.status = "login";
}

export async function setDisplayName(name: string): Promise<void> {
  const account = await updateAccount(name);
  auth.displayName = account.display_name ?? "";
}

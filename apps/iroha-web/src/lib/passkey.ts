import {
  beginPasskeyLogin,
  beginPasskeyRegistration,
  finishPasskeyLogin,
  finishPasskeyRegistration,
  type AuthSession,
  type Passkey,
} from "./api";

// Passkeys use the browser's native WebAuthn JSON helpers (Safari 18+,
// Chrome 129+, Firefox 119+), so no client library is needed.
export function passkeysSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof PublicKeyCredential !== "undefined" &&
    typeof PublicKeyCredential.parseCreationOptionsFromJSON === "function" &&
    typeof PublicKeyCredential.parseRequestOptionsFromJSON === "function"
  );
}

// Thrown when the person closes or cancels the browser's passkey sheet.
export class PasskeyCancelled extends Error {}

function rethrow(cause: unknown): never {
  if (cause instanceof DOMException && cause.name === "NotAllowedError") {
    throw new PasskeyCancelled("Passkey request was cancelled.");
  }
  throw cause;
}

export async function registerPasskey(name: string): Promise<Passkey> {
  const options = await beginPasskeyRegistration();
  let credential: Credential | null;
  try {
    credential = await navigator.credentials.create({
      publicKey: PublicKeyCredential.parseCreationOptionsFromJSON(
        options.publicKey,
      ),
    });
  } catch (cause) {
    rethrow(cause);
  }
  if (!(credential instanceof PublicKeyCredential)) {
    throw new PasskeyCancelled("No passkey was created.");
  }
  return finishPasskeyRegistration(name, credential.toJSON());
}

export async function signInWithPasskey(): Promise<AuthSession> {
  const options = await beginPasskeyLogin();
  let credential: Credential | null;
  try {
    credential = await navigator.credentials.get({
      publicKey: PublicKeyCredential.parseRequestOptionsFromJSON(
        options.publicKey,
      ),
    });
  } catch (cause) {
    rethrow(cause);
  }
  if (!(credential instanceof PublicKeyCredential)) {
    throw new PasskeyCancelled("No passkey was selected.");
  }
  return finishPasskeyLogin(credential.toJSON());
}

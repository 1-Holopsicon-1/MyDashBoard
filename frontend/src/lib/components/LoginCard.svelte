<script lang="ts">
	import { authStatus } from '$lib/stores';
	import { fetchAuthStatus, authRegisterBegin, authRegisterFinish, authLoginBegin, authLoginFinish, authLogout, authAddKeyBegin, authAddKeyFinish } from '$lib/api';
	import type { RegistrationCredentialJSON, AuthenticationCredentialJSON } from '$lib/types';

	let { showAuth = false }: { showAuth?: boolean } = $props();
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function checkStatus() {
		try {
			const status = await fetchAuthStatus();
			authStatus.set(status);
		} catch {
			// ignore
		}
	}

	async function register() {
		loading = true;
		error = null;
		try {
			const resp = await authRegisterBegin();
			const pkOptions = preDecode((resp as unknown as Record<string, unknown>).publicKey);
			const credential = await navigator.credentials.create({ publicKey: pkOptions as unknown as PublicKeyCredentialCreationOptions });
			if (!credential) throw new Error('No credential returned');
			await authRegisterFinish(encodeRegistration(credential as PublicKeyCredential));
			await checkStatus();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Registration failed';
		} finally {
			loading = false;
		}
	}

	async function login() {
		loading = true;
		error = null;
		try {
			const resp = await authLoginBegin();
			const pkOptions = preDecode((resp as unknown as Record<string, unknown>).publicKey);
			const credential = await navigator.credentials.get({ publicKey: pkOptions as unknown as PublicKeyCredentialRequestOptions });
			if (!credential) throw new Error('No credential returned');
			await authLoginFinish(encodeLogin(credential as PublicKeyCredential));
			await checkStatus();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Login failed';
		} finally {
			loading = false;
		}
	}

	async function logout() {
		try {
			await authLogout();
			await checkStatus();
		} catch {
			// ignore
		}
	}

	async function addKey() {
		loading = true;
		error = null;
		try {
			const resp = await authAddKeyBegin();
			const pkOptions = preDecode((resp as unknown as Record<string, unknown>).publicKey);
			const credential = await navigator.credentials.create({ publicKey: pkOptions as unknown as PublicKeyCredentialCreationOptions });
			if (!credential) throw new Error('No credential returned');
			await authAddKeyFinish(encodeRegistration(credential as PublicKeyCredential));
			await checkStatus();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Add key failed';
		} finally {
			loading = false;
		}
	}

	function preDecode(opts: unknown): Record<string, unknown> {
		const o = opts as Record<string, unknown>;
		const result: Record<string, unknown> = { ...o };

		if (typeof o.challenge === 'string') {
			result.challenge = b64ToBuf(o.challenge);
		}

		if (o.user && typeof (o.user as Record<string, unknown>).id === 'string') {
			result.user = { ...(o.user as Record<string, unknown>), id: b64ToBuf((o.user as Record<string, unknown>).id as string) };
		}

		if (Array.isArray(o.excludeCredentials)) {
			result.excludeCredentials = o.excludeCredentials.map((c: Record<string, unknown>) => ({
				...c, id: b64ToBuf(c.id as string),
			}));
		}

		if (Array.isArray(o.allowCredentials)) {
			result.allowCredentials = o.allowCredentials.map((c: Record<string, unknown>) => ({
				...c, id: b64ToBuf(c.id as string),
			}));
		}

		return result;
	}

	function encodeRegistration(cred: PublicKeyCredential): RegistrationCredentialJSON {
		const r = cred.response as AuthenticatorAttestationResponse;
		return {
			id: cred.id,
			rawId: bufToB64(cred.rawId),
			type: cred.type,
			response: {
				attestationObject: bufToB64(r.attestationObject),
				clientDataJSON: bufToB64(r.clientDataJSON),
			},
		};
	}

	function encodeLogin(cred: PublicKeyCredential): AuthenticationCredentialJSON {
		const r = cred.response as AuthenticatorAssertionResponse;
		return {
			id: cred.id,
			rawId: bufToB64(cred.rawId),
			type: cred.type,
			response: {
				authenticatorData: bufToB64(r.authenticatorData),
				clientDataJSON: bufToB64(r.clientDataJSON),
				signature: bufToB64(r.signature),
				userHandle: r.userHandle ? bufToB64(r.userHandle) : null,
			},
		};
	}

	function bufToB64(buf: ArrayBuffer): string {
		const bytes = new Uint8Array(buf);
		let binary = '';
		for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
		return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
	}

	function b64ToBuf(b64: string): ArrayBuffer {
		let s = b64.replace(/-/g, '+').replace(/_/g, '/');
		while (s.length % 4) s += '=';
		const bin = atob(s);
		const buf = new Uint8Array(bin.length);
		for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i);
		return buf.buffer;
	}
</script>

{#if showAuth}
<div class="auth-panel">
	{#if !$authStatus.registered}
		<div class="auth-form">
			<h3>Register Passkey</h3>
			<button onclick={register} disabled={loading}>
				{loading ? 'Waiting...' : 'Register'}
			</button>
		</div>
	{:else if !$authStatus.authenticated}
		<div class="auth-form">
			<h3>Authenticate</h3>
			<p class="hint">Use your passkey to access full dashboard</p>
			<button onclick={login} disabled={loading}>
				{loading ? 'Waiting...' : 'Login with Passkey'}
			</button>
		</div>
	{:else}
		<div class="auth-info">
			<span class="auth-badge">authenticated</span>
			<button class="add-key-btn" onclick={addKey} disabled={loading}>Add Key</button>
			<button class="logout-btn" onclick={logout}>Logout</button>
		</div>
	{/if}

	{#if error}
		<p class="error">{error}</p>
	{/if}
</div>
{/if}

<style>
	.auth-panel {
		padding: 12px 16px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius, 0);
		background: var(--color-card);
	}

	.auth-form {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	h3 {
		font-family: var(--font-mono);
		font-size: 0.85rem;
		color: var(--color-text);
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}

	.hint {
		font-size: 0.75rem;
		color: var(--color-text-secondary);
	}

	button {
		padding: 8px 16px;
		border: 1px solid var(--color-accent);
		border-radius: var(--radius, 0);
		background: transparent;
		color: var(--color-accent);
		font-family: var(--font-mono);
		font-size: 0.8rem;
		cursor: pointer;
		transition: all 0.2s ease;
		text-transform: uppercase;
	}

	button:hover:not(:disabled) {
		background: var(--color-accent);
		color: var(--color-bg);
	}

	button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.auth-info {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.auth-badge {
		font-family: var(--font-mono);
		font-size: 0.65rem;
		padding: 3px 8px;
		color: var(--color-online);
		border: 1px solid var(--color-online);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.add-key-btn {
		padding: 4px 10px;
		border: 1px solid var(--color-accent);
		font-size: 0.7rem;
		color: var(--color-accent);
	}

	.logout-btn {
		padding: 4px 10px;
		border: 1px solid var(--color-border);
		font-size: 0.7rem;
		color: var(--color-text-secondary);
	}

	.logout-btn:hover {
		border-color: var(--color-offline);
		color: var(--color-offline);
		background: transparent;
	}

	.error {
		margin-top: 8px;
		color: var(--color-offline);
		font-family: var(--font-mono);
		font-size: 0.75rem;
	}
</style>

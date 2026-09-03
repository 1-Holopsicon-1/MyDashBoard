import type { TailscaleDevice, ServiceStatus, ContainerStatus, SimplexLink, AuthStatus, RegistrationCredentialJSON, AuthenticationCredentialJSON } from './types';

const BASE = '/api';

async function fetchJSON<T>(path: string, options?: RequestInit): Promise<T> {
	const res = await fetch(`${BASE}${path}`, options);
	if (!res.ok) {
		throw new Error(`API error: ${res.status} ${res.statusText}`);
	}
	const ct = res.headers.get('content-type') ?? '';
	if (!ct.includes('application/json')) {
		throw new Error(`Expected JSON, got ${ct}`);
	}
	return res.json() as Promise<T>;
}

export function fetchDevices(signal?: AbortSignal): Promise<TailscaleDevice[]> {
	return fetchJSON<TailscaleDevice[]>('/tailscale/devices', { signal });
}

export function fetchServices(signal?: AbortSignal): Promise<ServiceStatus[]> {
	return fetchJSON<ServiceStatus[]>('/services', { signal });
}

export function fetchContainers(signal?: AbortSignal): Promise<ContainerStatus[]> {
	return fetchJSON<ContainerStatus[]>('/containers', { signal });
}

export function fetchSimplexLinks(signal?: AbortSignal): Promise<SimplexLink[]> {
	return fetchJSON<SimplexLink[]>('/simplex/links', { signal });
}

export function fetchAuthStatus(signal?: AbortSignal): Promise<AuthStatus> {
	return fetchJSON<AuthStatus>('/auth/status', { signal });
}

export function authRegisterBegin(signal?: AbortSignal): Promise<PublicKeyCredentialCreationOptionsJSON> {
	return fetchJSON<PublicKeyCredentialCreationOptionsJSON>('/auth/register-begin', { method: 'POST', signal });
}

export async function authRegisterFinish(credential: RegistrationCredentialJSON, signal?: AbortSignal): Promise<void> {
	await fetchJSON('/auth/register-finish', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(credential),
		signal,
	});
}

export function authLoginBegin(signal?: AbortSignal): Promise<PublicKeyCredentialRequestOptionsJSON> {
	return fetchJSON<PublicKeyCredentialRequestOptionsJSON>('/auth/login-begin', { method: 'POST', signal });
}

export async function authLoginFinish(credential: AuthenticationCredentialJSON, signal?: AbortSignal): Promise<void> {
	await fetchJSON('/auth/login-finish', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(credential),
		signal,
	});
}

export async function authLogout(signal?: AbortSignal): Promise<void> {
	await fetchJSON('/auth/logout', { method: 'POST', signal });
}

export function authAddKeyBegin(signal?: AbortSignal): Promise<PublicKeyCredentialCreationOptionsJSON> {
	return fetchJSON<PublicKeyCredentialCreationOptionsJSON>('/auth/add-key-begin', { method: 'POST', signal });
}

export async function authAddKeyFinish(credential: RegistrationCredentialJSON, signal?: AbortSignal): Promise<void> {
	await fetchJSON('/auth/add-key-finish', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(credential),
		signal,
	});
}
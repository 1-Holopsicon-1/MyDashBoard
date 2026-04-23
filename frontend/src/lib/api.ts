import type { TailscaleDevice, ServiceStatus, ContainerStatus, SimplexLink, AuthStatus } from './types';

const BASE = '/api';

async function fetchJSON<T>(path: string, options?: RequestInit): Promise<T> {
	const res = await fetch(`${BASE}${path}`, options);
	if (!res.ok) {
		throw new Error(`API error: ${res.status} ${res.statusText}`);
	}
	return res.json();
}

export function fetchDevices(): Promise<TailscaleDevice[]> {
	return fetchJSON<TailscaleDevice[]>('/tailscale/devices');
}

export function fetchServices(): Promise<ServiceStatus[]> {
	return fetchJSON<ServiceStatus[]>('/services');
}

export function fetchContainers(): Promise<ContainerStatus[]> {
	return fetchJSON<ContainerStatus[]>('/containers');
}

export function fetchSimplexLinks(): Promise<SimplexLink[]> {
	return fetchJSON<SimplexLink[]>('/simplex/links');
}

export function fetchAuthStatus(): Promise<AuthStatus> {
	return fetchJSON<AuthStatus>('/auth/status');
}

export function authRegisterBegin(): Promise<PublicKeyCredentialCreationOptionsJSON> {
	return fetchJSON<PublicKeyCredentialCreationOptionsJSON>('/auth/register-begin', { method: 'POST' });
}

export async function authRegisterFinish(credential: unknown): Promise<void> {
	await fetchJSON('/auth/register-finish', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(credential),
	});
}

export function authLoginBegin(): Promise<PublicKeyCredentialRequestOptionsJSON> {
	return fetchJSON<PublicKeyCredentialRequestOptionsJSON>('/auth/login-begin', { method: 'POST' });
}

export async function authLoginFinish(credential: unknown): Promise<void> {
	await fetchJSON('/auth/login-finish', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(credential),
	});
}

export async function authLogout(): Promise<void> {
	await fetchJSON('/auth/logout', { method: 'POST' });
}

export function authAddKeyBegin(): Promise<PublicKeyCredentialCreationOptionsJSON> {
	return fetchJSON<PublicKeyCredentialCreationOptionsJSON>('/auth/add-key-begin', { method: 'POST' });
}

export async function authAddKeyFinish(credential: unknown): Promise<void> {
	await fetchJSON('/auth/add-key-finish', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(credential),
	});
}

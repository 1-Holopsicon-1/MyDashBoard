import { writable } from 'svelte/store';
import type { TailscaleDevice, ServiceStatus, ContainerStatus, SimplexLink, AuthStatus, Theme } from './types';

export const devices = writable<TailscaleDevice[]>([]);
export const services = writable<ServiceStatus[]>([]);
export const containers = writable<ContainerStatus[]>([]);
export const simplexLinks = writable<SimplexLink[]>([]);
export const devicesLoading = writable(true);
export const servicesLoading = writable(true);
export const containersLoading = writable(true);
export const simplexLoading = writable(true);
export const devicesError = writable<string | null>(null);
export const servicesError = writable<string | null>(null);
export const containersError = writable<string | null>(null);
export const simplexError = writable<string | null>(null);
export const lastUpdated = writable<Date | null>(null);
export const authStatus = writable<AuthStatus>({ registered: false, authenticated: false });

function getInitialTheme(): Theme {
	if (typeof window !== 'undefined') {
		const saved = localStorage.getItem('theme');
		if (saved === 'terminal' || saved === 'minimal' || saved === 'glass') {
			return saved;
		}
	}
	return 'terminal';
}

export const theme = writable<Theme>(getInitialTheme());

theme.subscribe((value) => {
	if (typeof window !== 'undefined') {
		localStorage.setItem('theme', value);
	}
});

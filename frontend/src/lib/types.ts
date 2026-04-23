export interface TailscaleDevice {
	id: string;
	name: string;
	hostname: string;
	os: string;
	addresses?: string[];
	lastSeen: string;
	connectedToControl: boolean;
	authorized: boolean;
}

export interface ServiceStatus {
	name: string;
	url: string;
	online: boolean;
	latency_ms: number;
	code: number;
}

export interface ContainerStatus {
	name: string;
	image: string;
	state: string;
	status: string;
	online: boolean;
}

export interface SimplexLink {
	container: string;
	address: string;
}

export interface AuthStatus {
	registered: boolean;
	authenticated: boolean;
}

export type Theme = 'terminal' | 'minimal' | 'glass';

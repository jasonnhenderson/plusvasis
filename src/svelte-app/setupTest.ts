// source: https://github.com/davipon/svelte-component-test-recipes/blob/main/setupTest.ts

/* eslint-disable @typescript-eslint/no-empty-function */
import type { Navigation, Page } from '@sveltejs/kit';
import matchers from '@testing-library/jest-dom/matchers';
import { readable } from 'svelte/store';
import { expect, vi } from 'vitest';

import * as environment from '$app/environment';
import * as navigation from '$app/navigation';
import * as stores from '$app/stores';

expect.extend(matchers);

// Mock SvelteKit runtime module $app/environment
vi.mock('$app/environment', (): typeof environment => ({
	browser: false,
	dev: true,
	building: false,
	version: 'any'
}));

// Mock SvelteKit runtime module $app/navigation
vi.mock('$app/navigation', (): typeof navigation => ({
	afterNavigate: () => {},
	beforeNavigate: () => {},
	disableScrollHandling: () => {},
	goto: () => Promise.resolve(),
	invalidate: () => Promise.resolve(),
	invalidateAll: () => Promise.resolve(),
	preloadData: () => Promise.resolve(),
	preloadCode: () => Promise.resolve()
}));

// Mock SvelteKit runtime module $app/stores
vi.mock('$app/stores', (): typeof stores => {
	const getStores: typeof stores.getStores = () => {
		const navigating = readable<Navigation | null>(null);
		const page = readable<Page>({
			url: new URL('http://localhost'),
			params: {},
			route: {
				id: null
			},
			status: 200,
			error: null,
			data: {},
			form: undefined
		});
		const updated = {
			...readable(false),
			check: async () => false
		};

		return { navigating, page, updated };
	};

	const page: typeof stores.page = {
		subscribe(fn) {
			return getStores().page.subscribe(fn);
		}
	};
	const navigating: typeof stores.navigating = {
		subscribe(fn) {
			return getStores().navigating.subscribe(fn);
		}
	};
	const updated: typeof stores.updated = {
		...readable(false),
		check: async () => false
	};

	return {
		getStores,
		navigating,
		page,
		updated
	};
});

// Custom mocks

window.matchMedia =
	window.matchMedia ||
	function () {
		return {
			addListener: function () {
				return;
			}
		};
	};

class CanvasRenderingContext2DMock {
	createLinearGradient() {
		return {};
	}
}
Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
	value: () => new CanvasRenderingContext2DMock()
});

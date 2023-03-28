import type { ComponentType } from 'svelte';

export type Job = {
	user: string | null;
	containerName: string;
	dockerImage: string;
	shell: string;
	volumes: [string, string][];
	env: [string, string][];
	port: number;
	expose: boolean;
};

type JobField = {
	title: string;
	placeholder: string;
	info: string;
};

export const JobFields: {key: string, value: JobField}[] = [
	{
	  	key: 'containerName',
	  	value: {
			title: 'Container Name',
			placeholder: 'alpine',
			info: 'This is used to identify the container.',
		},
	},
	{
		key: 'dockerImage',
		value: {
			title: 'Docker Image',
			placeholder: 'alpine:latest',
			info: 'Images will be pulled from a registry, e.g. Docker Hub.',
		},
	},
	{
		key: 'shell',
		value: {
			title: 'Shell',
			placeholder: '/bin/sh',
			info: 'The shell to use when executing commands within the container.',
		},
	},
	{
		key: 'volumes',
		value: {
			title: 'Volumes',
			placeholder: 'docker_volume:/mnt/volume',
			info: 'For persistant storage, volumes are required. You can add multiple volumes by separating them with a comma.',
		},
	},
	{
		key: 'env',
		value: {
			title: 'Environment Variables',
			placeholder: 'ENV_VAR=VALUE',
			info: 'These will be passed to the container. You can add multiple environment variables by separating them with a comma.',
		},
	},
	{
		key: 'port',
		value: {
			title: 'Port',
			placeholder: '8080',
			info: 'The port to expose to the host.',
		},
	},
];

export type Tab = {
	name: string;
	component: ComponentType;
};

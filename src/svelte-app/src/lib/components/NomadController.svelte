<script lang="ts">
	import ExecController from '$lib/components/ExecController.svelte';
	import type { Job } from '$lib/types/Types';

	import { token } from '../../stores/auth';
	import { hostname } from '../../stores/environmentStore';
	import { currJob, currJobId, currJobStopped } from '../../stores/nomadStore';

	let execControllerComponent: ExecController;
	let wsUrl: string;

	let jobId: string;
	let job: Job;
	let isStopped: boolean;
	let authToken: string | undefined;
	currJobId.subscribe((value) => {
		jobId = value;
	});
	currJob.subscribe((value) => {
		job = value;
	});
	currJobStopped.subscribe((value) => {
		isStopped = value;
	});
	token.subscribe((value) => {
		authToken = value;
	});

	function setExecUrl() {
		const url = new URL(`${hostname}/job/${jobId}/exec`);
		url.protocol = url.protocol.replace('http', 'ws');
		url.searchParams.append('command', `["${job.shell}"]`);

		if (authToken) url.searchParams.append('access_token', authToken);

		wsUrl = url.toString();
	}

	$: if (!isStopped && jobId && job && authToken) {
		setExecUrl();
	}
</script>

<ExecController bind:this={execControllerComponent} {wsUrl} />

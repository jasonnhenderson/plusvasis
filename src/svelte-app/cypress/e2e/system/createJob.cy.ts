/* eslint-disable cypress/unsafe-to-chain-command */
/* eslint-disable cypress/no-unnecessary-waiting */

Cypress.on('window:before:load', (win) => {
	cy.stub(win.console, 'log').as('consoleLog');
});

describe('createJob spec', () => {
	it('User creates a new job', () => {
		cy.intercept('POST', 'https://api.plusvasis.xyz/jobs').as('jobsCreated');
		cy.intercept('GET', '/__data.json?x-sveltekit-invalidated=1_1').as('jobs');

		cy.visit('https://app.plusvasis.xyz/login');

		cy.wait(1500);

		cy.get('input[name="email"]').type('example@email.com');
		cy.get('input[name="password"]').type('th1s1sJustATest');
		cy.get('button[type="submit"]').click();

		cy.visit('https://app.plusvasis.xyz/create');

		// Wait for a few seconds (e.g., 3 seconds) before proceeding
		cy.wait(1500);

		cy.get('input[name="containerName"]', { timeout: 10000 })
			.should('be.visible')
			.type('cypress-test', { delay: 100 })
			.should('have.value', 'cypress-test');

		cy.get('input[name="dockerImage"]', { timeout: 10000 })
			.should('be.visible')
			.type('nginx', { delay: 100 })
			.should('have.value', 'nginx');

		cy.get('button[type="submit"]').click();

		cy.wait('@jobsCreated');
		cy.get('@consoleLog').should('be.calledWith', 'Container Created');

		cy.wait('@jobs');
		cy.get('div[data-testid="job-list"]').should('exist');
	});
});

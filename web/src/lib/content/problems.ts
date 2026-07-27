/**
 * Copy for the page that answers "what is wrong?".
 *
 * The hardest line here is the empty state. "No problems" has to read as a
 * finding rather than as a page that failed to load — somebody arriving here
 * worried needs to be told plainly that nothing is wrong, not shown a blank
 * space they will interpret as a bug.
 */
export const PROBLEMS_COPY = {
	title: 'What needs attention',
	description:
		'Anything currently wrong, and anything that failed while you were not watching. ' +
		'This is the page to open when something seems off.',

	alertsTitle: 'Happening now',
	alertsHint: 'These clear themselves as soon as the problem stops.',

	failuresTitle: 'Runs that did not finish',
	failuresHint:
		'These stay here until the job is run again. A failed attempt never replaces ' +
		'something that was already working.',

	/** The whole page is empty. */
	allClearTitle: 'Nothing is wrong',
	allClearBody:
		'No alerts, and nothing has failed. Your servers are being checked every ' +
		'few minutes, so this page fills itself in if that changes.',

	/** One half is empty while the other is not, so it stays quiet. */
	noAlerts: 'Nothing is wrong right now.',
	noFailures: 'Nothing has failed.',

	openJob: 'See what happened',
	openServer: 'Go to the server',

	/** Said once, at the bottom, rather than on every row. */
	scheduledNote:
		'Runs marked Scheduled had nobody watching them. Those are the ones worth ' +
		'reading first — you would already know about the others.'
};

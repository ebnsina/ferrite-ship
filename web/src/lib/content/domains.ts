/**
 * Copy for pointing a domain at a server.
 *
 * Kept out of the component because it is the part that has to be got right:
 * DNS is where people get stuck, and the difference between "add an A record"
 * and a sentence that says which record, what name, and what value is the
 * difference between this working first time and a support conversation.
 */
export const DOMAIN_COPY = {
	title: 'Where this server answers on the web',
	intro:
		'Give it a domain and everything you install gets its own web address, with ' +
		'certificates handled for you. Without one, tools stay private to the server ' +
		'and you reach them through a secure tunnel.',

	/** Written as an instruction someone can follow at their DNS provider. */
	dnsTitle: 'First, point the domain here',
	dnsSteps: (ip: string) => [
		`At whoever you bought the domain from, add an A record.`,
		`Name it * — the asterisk on its own, which covers every address under the domain.`,
		`Point it at ${ip}.`,
		`Changes can take a few minutes to travel, and occasionally an hour.`
	],

	domainLabel: 'Domain',
	domainHint: 'Just the name, like example.com. A tool called Grafana would then be at grafana.example.com.',

	emailLabel: 'Email for certificate notices',
	emailHint:
		'Certificates are issued in your name and renew on their own. This is where ' +
		'you are told if one ever cannot be.',

	save: 'Save domain',
	saving: 'Saving…',
	clear: 'Remove domain',

	/** Shown once a domain is set, so the result is visible rather than implied. */
	live: (domain: string) => `Tools installed here will be reachable under ${domain}.`,

	/**
	 * Said plainly rather than hidden, because it is the one consequence people
	 * do not expect: the certificate is requested when a tool is installed, and
	 * it fails if the record is not there yet.
	 */
	note:
		'Nothing is checked when you save this. The domain is used the next time you ' +
		'install something, and that is when a wrong record shows up.'
};

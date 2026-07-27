/**
 * The help content.
 *
 * Every answer here describes something the product actually does today. When
 * a feature does not exist, the answer says so plainly rather than describing
 * how it will work — a help page that promises things is worse than no help
 * page, because it sends people looking for a button that is not there.
 */
export interface HelpTopic {
	question: string;
	answer: string;
}

export interface HelpSection {
	heading: string;
	topics: HelpTopic[];
}

export const helpSections: HelpSection[] = [
	{
		heading: 'Getting a server ready',
		topics: [
			{
				question: 'What does "set up" actually change?',
				answer:
					'It installs available updates, creates an everyday login account, turns on a firewall that closes everything except the ports you use, blocks repeated failed logins, switches on automatic security updates, sets the clock, adds swap space and applies some sensible network settings.'
			},
			{
				question: 'Is it safe to run setup twice?',
				answer:
					'Yes, and it is meant to be. Every step checks whether the work is already done before doing anything, so a second run reports that nothing needed changing. That is also how you repair a server that has drifted.'
			},
			{
				question: 'Will it lock me out?',
				answer:
					'It refuses to. Password logins are only turned off when a login key is already installed on the machine, and never when Ferrite Ship itself is signing in with a password — it would lock out both of us. When it skips that step it tells you why.'
			},
			{
				question: 'What does "Check" do?',
				answer:
					'It runs every check and reports what would change, without touching the machine. Use it when you want to know what setup would do before letting it.'
			}
		]
	},
	{
		heading: 'Databases and other tools',
		topics: [
			{
				question: 'Where do the databases actually run?',
				answer:
					'As containers on your server, each with its own folder under /opt/ferrite that you can read. Nothing is hidden from you — the file describing what runs is plain text on your own machine.'
			},
			{
				question: 'Why can I not connect to my database from my laptop?',
				answer:
					'Because it only listens on the server itself. A database open to the internet is found by scanners within minutes of appearing. Open the tunnel command shown on the tool’s page, leave it running, and connect to 127.0.0.1 as if the database were local.'
			},
			{
				question: 'Can I run queries without leaving here?',
				answer:
					'Yes. Each installed database has a console on its page. PostgreSQL and ClickHouse take SQL; Redis takes commands. There are ready-made queries to start from, and you can keep your own.'
			},
			{
				question: 'What happens to my data if I remove a tool?',
				answer:
					'It is kept, unless you tick the box that says otherwise. Removing stops the tool and deletes its settings; the stored data stays where it is and comes back if you install the tool again.'
			}
		]
	},
	{
		heading: 'Backups',
		topics: [
			{
				question: 'Where do backups go?',
				answer:
					'To storage you choose in Settings — anything that speaks S3, including Amazon, Cloudflare R2, Backblaze B2 and DigitalOcean Spaces. Deliberately not the server itself: a copy on the same disk survives a mistake, but not the disk.'
			},
			{
				question: 'Do backups happen automatically?',
				answer:
					'Not yet. Today you take them when you choose to, from the tool’s page. Scheduling is not built.'
			},
			{
				question: 'Which tools can be backed up?',
				answer:
					'PostgreSQL, Redis and ClickHouse. MediaMTX keeps no data, so there is nothing to copy.'
			},
			{
				question: 'What does restoring do?',
				answer:
					'It replaces everything currently in that tool with the contents of the backup. Anything written since the backup was taken is lost, and there is no undo — so take a fresh backup first if you are unsure.'
			}
		]
	},
	{
		heading: 'Your own applications',
		topics: [
			{
				question: 'How does deploying work?',
				answer:
					'Give us a git repository and a branch. If it has a Dockerfile we build with that. If it does not, we work out how to build it — Rust, Go, Python and Node are all understood. Then it runs on your server and, if you gave it a domain, it is served over HTTPS.'
			},
			{
				question: 'Can I deploy a private repository?',
				answer:
					'Not yet. Repositories must be publicly readable, because there is no deploy key to give the server access to a private one.'
			},
			{
				question: 'How do certificates work?',
				answer:
					'Automatically, from Let’s Encrypt, as long as the domain’s DNS points at the server before you deploy. Nothing to renew and nothing to remember.'
			},
			{
				question: 'How does my application reach a database on the same server?',
				answer:
					'Take the connection details from the tool’s page and add them as environment variables on the application. Both are on the same machine, so the address is the server’s own.'
			}
		]
	},
	{
		heading: 'Access and safety',
		topics: [
			{
				question: 'What do you store about my servers?',
				answer:
					'The address, the account name, and the sign-in details — encrypted before they are written down. Passwords and keys are never shown again, never written to a log, and never sent back to your browser.'
			},
			{
				question: 'How do you know you are talking to the right server?',
				answer:
					'The server’s identity is recorded the first time it is reached and checked on every connection after. If it ever changes, Ferrite Ship refuses to connect rather than carrying on.'
			},
			{
				question: 'Something failed. Where do I look?',
				answer:
					'History shows every run. Open one and you will see each step, whether it changed anything, and the exact commands and output — including the reason it stopped.'
			}
		]
	}
];

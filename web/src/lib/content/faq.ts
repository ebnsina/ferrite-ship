export interface FaqEntry {
	question: string;
	answer: string;
}

export const faqEntries: FaqEntry[] = [
	{
		question: 'Do I need to know Linux?',
		answer:
			'No. Everything has a button, and we explain what each one does in plain words. If you already know your way around, nothing stops you working the way you always have.'
	},
	{
		question: 'Which servers work with this?',
		answer:
			'Any server running Ubuntu 22.04 or 24.04, from any provider — Hetzner, DigitalOcean, Vultr, AWS, or a machine under your desk. You buy the server, we look after it.'
	},
	{
		question: 'Do you keep my server password?',
		answer:
			'No. During setup your server creates its own key and uses that to check in with us. There is no master password stored on our side, so there is nothing to steal.'
	},
	{
		question: 'What if my server is already set up?',
		answer:
			'Connect it anyway. We check what is already done and leave it alone — running our setup twice changes nothing the second time.'
	},
	{
		question: 'What happens if your service goes down?',
		answer:
			'Your servers and your websites keep running exactly as before. We are the control panel, not the thing your visitors talk to.'
	},
	{
		question: 'Can I still log in myself?',
		answer:
			'Yes, always. It is your server. You can log in the usual way whenever you want, and we will show you anything that changed outside the dashboard.'
	},
	{
		question: 'How do I remove it?',
		answer:
			'One command takes us off the machine completely and leaves everything running. No lock-in, no leftovers.'
	}
];

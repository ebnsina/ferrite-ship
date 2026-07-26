export interface PricingPlan {
	id: string;
	name: string;
	/** Monthly price in whole currency units. `null` renders as "Let's talk". */
	monthlyPrice: number | null;
	tagline: string;
	features: string[];
	featured?: boolean;
	ctaLabel: string;
}

/**
 * PLACEHOLDER PRICING — the tiers and limits are a sensible starting shape,
 * but the numbers are guesses. Replace them before this page goes live.
 */
export const pricingCurrency = 'USD';

export const pricingPlans: PricingPlan[] = [
	{
		id: 'free',
		name: 'Starter',
		monthlyPrice: 0,
		tagline: 'For your first server.',
		features: [
			'1 server',
			'Setup and safety checks',
			'Browser terminal and files',
			'Community support'
		],
		ctaLabel: 'Start for free'
	},
	{
		id: 'pro',
		name: 'Pro',
		monthlyPrice: 19,
		tagline: 'For people running real things.',
		features: [
			'Up to 10 servers',
			'One-click tools and databases',
			'Automatic backups',
			'Alerts by email and chat',
			'Email support'
		],
		featured: true,
		ctaLabel: 'Start free trial'
	},
	{
		id: 'team',
		name: 'Team',
		monthlyPrice: 59,
		tagline: 'For agencies and teams.',
		features: [
			'Unlimited servers',
			'Invite your teammates',
			'Roles and permissions',
			'Full activity record',
			'Priority support'
		],
		ctaLabel: 'Start free trial'
	}
];

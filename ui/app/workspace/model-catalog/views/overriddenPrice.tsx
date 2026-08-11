import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { formatTokenPriceCompact, formatTokenPriceFull } from "@/lib/utils/numbers";

interface OverriddenPriceProps {
	/** Catalog price from governance_model_pricing. */
	base?: number;
	/** Post-override price. Undefined means this field is not overridden. */
	override?: number;
	variant: "compact" | "full";
	/** Name of the override that produced `override`, shown in the tooltip. */
	overrideName?: string;
	testId?: string;
}

/**
 * Renders a catalog price, striking it through and showing the effective price
 * beside it when a pricing override changes that field. With no override it
 * renders exactly what the plain formatter would, so unaffected rows are
 * visually unchanged.
 */
export default function OverriddenPrice({ base, override, variant, overrideName, testId }: OverriddenPriceProps) {
	const format = variant === "compact" ? formatTokenPriceCompact : formatTokenPriceFull;

	if (override === undefined || override === null) {
		return <span data-testid={testId}>{format(base)}</span>;
	}

	return (
		<TooltipProvider>
			<Tooltip>
				<TooltipTrigger asChild>
					<span className="inline-flex flex-wrap items-baseline justify-end gap-x-1.5" data-testid={testId}>
						<span className="text-muted-foreground text-xs line-through" data-testid={testId ? `${testId}-original` : undefined}>
							{format(base)}
						</span>
						<span data-testid={testId ? `${testId}-override` : undefined}>{format(override)}</span>
					</span>
				</TooltipTrigger>
				<TooltipContent>{overrideName ? `Overridden by "${overrideName}"` : "Overridden by a custom pricing override"}</TooltipContent>
			</Tooltip>
		</TooltipProvider>
	);
}
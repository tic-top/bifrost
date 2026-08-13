import { Button } from "@/components/ui/button";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { getErrorMessage, setProviderFormDirtyState, useAppDispatch } from "@/lib/store";
import { useUpdateProviderMutation } from "@/lib/store/apis/providersApi";
import type { ModelProvider } from "@/lib/types/config";
import { openaiConfigFormSchema, type OpenAIConfigFormSchema } from "@/lib/types/schemas";
import { RbacOperation, RbacResource, useRbac } from "@enterprise/lib";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { buildProviderUpdatePayload } from "../views/utils";

interface OpenAIConfigFormFragmentProps {
	provider: ModelProvider;
}

export function OpenAIConfigFormFragment({ provider }: OpenAIConfigFormFragmentProps) {
	const dispatch = useAppDispatch();
	const hasUpdateProviderAccess = useRbac(RbacResource.ModelProvider, RbacOperation.Update);
	const [updateProvider, { isLoading: isUpdatingProvider }] = useUpdateProviderMutation();
	const form = useForm<OpenAIConfigFormSchema, any, OpenAIConfigFormSchema>({
		resolver: zodResolver(openaiConfigFormSchema) as Resolver<OpenAIConfigFormSchema, any, OpenAIConfigFormSchema>,
		mode: "onChange",
		reValidateMode: "onChange",
		defaultValues: {
			disable_store: provider.openai_config?.disable_store ?? false,
			strip_response_input_item_state: provider.openai_config?.strip_response_input_item_state ?? false,
			upstream_session_header: provider.openai_config?.upstream_session_header ?? "",
		},
	});

	useEffect(() => {
		dispatch(setProviderFormDirtyState(form.formState.isDirty));
	}, [form.formState.isDirty, dispatch]);

	useEffect(() => {
		form.reset({
			disable_store: provider.openai_config?.disable_store ?? false,
			strip_response_input_item_state: provider.openai_config?.strip_response_input_item_state ?? false,
			upstream_session_header: provider.openai_config?.upstream_session_header ?? "",
		});
	}, [form, provider.name, provider.openai_config]);

	const onSubmit = (data: OpenAIConfigFormSchema) => {
		updateProvider(
			buildProviderUpdatePayload(provider, {
				openai_config: {
					disable_store: data.disable_store,
					strip_response_input_item_state: data.strip_response_input_item_state,
					upstream_session_header: data.upstream_session_header.trim(),
				},
			}),
		)
			.unwrap()
			.then(() => {
				toast.success("OpenAI configuration updated successfully");
				form.reset(data);
			})
			.catch((err) => {
				toast.error("Failed to update OpenAI configuration", {
					description: getErrorMessage(err),
				});
			});
	};

	return (
		<Form {...form}>
			<form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 px-6" data-testid="provider-config-openai-content">
				<div className="space-y-4">
					<FormField
						control={form.control}
						name="disable_store"
						render={({ field }) => (
							<FormItem>
								<div className="flex items-center justify-between space-x-2">
									<div className="space-y-0.5">
										<FormLabel>Disable Store</FormLabel>
										<p className="text-muted-foreground text-xs">
											With the Responses API, store defaults to true, and when it is on, the generated response is stored for later
											retrieval via API. OpenAI exposes endpoints to retrieve and delete stored responses, so your response IDs become
											durable server-side objects instead of one-shot IDs.
										</p>
									</div>
									<FormControl>
										<Switch
											data-testid="provider-openai-disable-store-switch"
											size="md"
											checked={field.value}
											disabled={!hasUpdateProviderAccess}
											onCheckedChange={(checked) => {
												field.onChange(checked);
												form.trigger("disable_store");
											}}
										/>
									</FormControl>
								</div>
								<FormMessage />
							</FormItem>
						)}
					/>
					<FormField
						control={form.control}
						name="strip_response_input_item_state"
						render={({ field }) => (
							<FormItem>
								<div className="flex items-center justify-between space-x-2">
									<div className="space-y-0.5">
										<FormLabel>Strip Provider Replay State</FormLabel>
										<p className="text-muted-foreground text-xs">
											Remove provider-scoped item IDs and encrypted reasoning state when a stateless compatible gateway rejects replay.
										</p>
									</div>
									<FormControl>
										<Switch
											data-testid="provider-openai-strip-replay-state-switch"
											size="md"
											checked={field.value}
											disabled={!hasUpdateProviderAccess}
											onCheckedChange={field.onChange}
										/>
									</FormControl>
								</div>
								<FormMessage />
							</FormItem>
						)}
					/>
					<FormField
						control={form.control}
						name="upstream_session_header"
						render={({ field }) => (
							<FormItem>
								<FormLabel>Upstream Session Header</FormLabel>
								<p className="text-muted-foreground text-xs">
									Optional provider-owned header populated from Bifrost session identity for sticky routing and prompt-cache locality.
								</p>
								<FormControl>
									<Input placeholder="x-provider-session-id" disabled={!hasUpdateProviderAccess} {...field} />
								</FormControl>
								<FormMessage />
							</FormItem>
						)}
					/>
				</div>

				<div className="flex justify-end space-x-2 pb-6">
					<Button
						type="submit"
						disabled={!form.formState.isDirty || !form.formState.isValid || !hasUpdateProviderAccess || isUpdatingProvider}
						isLoading={isUpdatingProvider}
					>
						Save OpenAI Configuration
					</Button>
				</div>
			</form>
		</Form>
	);
}
export interface GroupUserConcurrencyFormState {
  user_concurrency_enabled: boolean;
  user_concurrency_limit: number;
}

export function applyGroupUserConcurrencyToEditForm(
  target: GroupUserConcurrencyFormState,
  source: GroupUserConcurrencyFormState,
): void {
  target.user_concurrency_enabled = source.user_concurrency_enabled ?? false;
  target.user_concurrency_limit = source.user_concurrency_limit ?? 1;
}

export function resetGroupUserConcurrencyCreateForm(
  target: GroupUserConcurrencyFormState,
): void {
  target.user_concurrency_enabled = false;
  target.user_concurrency_limit = 1;
}

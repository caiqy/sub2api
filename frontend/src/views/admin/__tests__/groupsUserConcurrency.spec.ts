import { describe, expect, it } from "vitest";

import {
  applyGroupUserConcurrencyToEditForm,
  resetGroupUserConcurrencyCreateForm,
} from "../groupsUserConcurrency";

describe("groupsUserConcurrency", () => {
  it("preserves backend zero value when hydrating edit form", () => {
    const target = {
      user_concurrency_enabled: true,
      user_concurrency_limit: 3,
    };

    applyGroupUserConcurrencyToEditForm(target, {
      user_concurrency_enabled: true,
      user_concurrency_limit: 0,
    });

    expect(target).toEqual({
      user_concurrency_enabled: true,
      user_concurrency_limit: 0,
    });
  });

  it("resets create form concurrency state to disabled with limit one", () => {
    const target = {
      user_concurrency_enabled: true,
      user_concurrency_limit: 7,
    };

    resetGroupUserConcurrencyCreateForm(target);

    expect(target).toEqual({
      user_concurrency_enabled: false,
      user_concurrency_limit: 1,
    });
  });
});

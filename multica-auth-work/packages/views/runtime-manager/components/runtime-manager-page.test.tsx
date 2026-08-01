import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { BindingList } from "./runtime-manager-page";

const bindings = [
  {
    id: "binding-1",
    runtime_id: "runtime-1",
    provider: "anthropic",
    state: "ready",
    home_ref: "home_opaque_1",
  },
  {
    id: "binding-2",
    runtime_id: "runtime-2",
    provider: "openai",
    state: "ready",
    home_ref: null,
  },
];

function Harness() {
  const [selected, setSelected] = useState("binding-1");
  return (
    <BindingList
      bindings={bindings}
      selectedBindingId={selected}
      onSelect={setSelected}
    />
  );
}

describe("Runtime Manager binding keyboard navigation", () => {
  it("exposes listbox selection and moves selection and focus with arrow keys", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    expect(screen.getByRole("listbox", { name: "Runtime bindings" })).toHaveAttribute(
      "aria-orientation",
      "vertical",
    );
    const options = screen.getAllByRole("option");
    expect(options[0]).toHaveAttribute("aria-selected", "true");
    expect(options[0]).toHaveAttribute("tabindex", "0");
    expect(options[1]).toHaveAttribute("aria-selected", "false");
    expect(options[1]).toHaveAttribute("tabindex", "-1");

    options[0]?.focus();
    await user.keyboard("{ArrowDown}");

    expect(options[1]).toHaveFocus();
    expect(options[1]).toHaveAttribute("aria-selected", "true");
    expect(options[1]).toHaveAttribute("tabindex", "0");
    expect(options[0]).toHaveAttribute("aria-selected", "false");

    await user.keyboard("{Home}");
    expect(options[0]).toHaveFocus();
    expect(options[0]).toHaveAttribute("aria-selected", "true");
  });

  it("activates an option with ordinary keyboard button semantics", async () => {
    const user = userEvent.setup();
    render(<Harness />);
    const options = screen.getAllByRole("option");

    options[1]?.focus();
    await user.keyboard("{Enter}");

    expect(options[1]).toHaveAttribute("aria-selected", "true");
    expect(options[1]).toHaveFocus();
  });
});

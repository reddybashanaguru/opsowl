import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { TaskStatusBadge } from "../TaskStatusBadge";

describe("TaskStatusBadge", () => {
  it.each([
    ["todo", "Todo", "text-slate-400"],
    ["in_progress", "In Progress", "text-violet-400"],
    ["done", "Done", "text-green-400"],
    ["cancelled", "Cancelled", "text-slate-500"],
  ])("renders correct label and color for status %s", (status, label, colorClass) => {
    render(<TaskStatusBadge status={status} />);
    const badge = screen.getByText(label);
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain(colorClass);
  });

  it("falls back gracefully for unknown status", () => {
    render(<TaskStatusBadge status="unknown_status" />);
    expect(screen.getByText("unknown_status")).toBeInTheDocument();
  });
});

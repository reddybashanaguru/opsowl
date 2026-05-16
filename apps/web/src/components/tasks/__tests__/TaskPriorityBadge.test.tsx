import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { TaskPriorityBadge } from "../TaskPriorityBadge";

describe("TaskPriorityBadge", () => {
  it.each([
    ["critical", "Critical", "text-red-400"],
    ["high", "High", "text-orange-400"],
    ["medium", "Medium", "text-yellow-400"],
    ["low", "Low", "text-slate-400"],
  ])("renders correct label and color for priority %s", (priority, label, colorClass) => {
    render(<TaskPriorityBadge priority={priority} />);
    const badge = screen.getByText(new RegExp(label, "i"));
    expect(badge).toBeInTheDocument();
    expect(badge.className).toContain(colorClass);
  });

  it("renders priority icon for critical", () => {
    render(<TaskPriorityBadge priority="critical" />);
    expect(screen.getByText("▲▲")).toBeInTheDocument();
  });

  it("renders priority icon for high", () => {
    render(<TaskPriorityBadge priority="high" />);
    expect(screen.getByText("▲")).toBeInTheDocument();
  });
});

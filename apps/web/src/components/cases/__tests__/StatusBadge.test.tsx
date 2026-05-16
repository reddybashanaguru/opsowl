import { render, screen } from "@testing-library/react";
import { StatusBadge } from "../StatusBadge";

describe("StatusBadge", () => {
  const cases = [
    { status: "draft",          label: "Draft" },
    { status: "open",           label: "Open" },
    { status: "in_progress",    label: "In Progress" },
    { status: "pending_review", label: "Pending Review" },
    { status: "closed",         label: "Closed" },
    { status: "escalated",      label: "Escalated" },
  ];

  for (const { status, label } of cases) {
    it(`renders "${label}" for status="${status}"`, () => {
      render(<StatusBadge status={status} />);
      expect(screen.getByText(label)).toBeInTheDocument();
    });
  }

  it("falls back to the raw status string for unknown values", () => {
    render(<StatusBadge status="unknown_status" />);
    expect(screen.getByText("unknown_status")).toBeInTheDocument();
  });

  it("applies md size classes when size=md", () => {
    const { container } = render(<StatusBadge status="open" size="md" />);
    expect(container.firstChild).toHaveClass("px-3");
  });

  it("applies sm size classes by default", () => {
    const { container } = render(<StatusBadge status="open" />);
    expect(container.firstChild).toHaveClass("px-2");
  });
});

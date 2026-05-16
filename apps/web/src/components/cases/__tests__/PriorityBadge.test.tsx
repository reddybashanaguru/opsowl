import { render, screen } from "@testing-library/react";
import { PriorityBadge } from "../PriorityBadge";

describe("PriorityBadge", () => {
  const priorities = ["critical", "high", "medium", "low"];

  for (const priority of priorities) {
    it(`renders capitalized label for priority="${priority}"`, () => {
      render(<PriorityBadge priority={priority} />);
      const expected = priority.charAt(0).toUpperCase() + priority.slice(1);
      expect(screen.getByText(expected)).toBeInTheDocument();
    });
  }

  it("renders a colored dot indicator", () => {
    const { container } = render(<PriorityBadge priority="critical" />);
    // the dot is a span with rounded-full
    const dot = container.querySelector("span > span");
    expect(dot).toHaveClass("rounded-full");
    expect(dot).toHaveClass("bg-red-400");
  });

  it("falls back gracefully for unknown priority", () => {
    render(<PriorityBadge priority="urgent" />);
    expect(screen.getByText("Urgent")).toBeInTheDocument();
  });
});

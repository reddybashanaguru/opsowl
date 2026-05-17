import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AIReviewBadge } from "../AIReviewBadge";

describe("AIReviewBadge", () => {
  it.each([
    ["pending",  "Pending Review"],
    ["approved", "Verified"],
    ["edited",   "Verified (Edited)"],
    ["rejected", "Rejected"],
  ])("renders correct label for status %s", (status, label) => {
    render(<AIReviewBadge status={status} />);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("falls back to pending for unknown status", () => {
    render(<AIReviewBadge status="unknown" />);
    expect(screen.getByText("Pending Review")).toBeInTheDocument();
  });

  it("has testid review-badge", () => {
    render(<AIReviewBadge status="pending" />);
    expect(screen.getByTestId("review-badge")).toBeInTheDocument();
  });

  it("applies amber classes for pending", () => {
    const { container } = render(<AIReviewBadge status="pending" />);
    expect(container.querySelector(".text-amber-300")).toBeInTheDocument();
  });

  it("applies green classes for approved", () => {
    const { container } = render(<AIReviewBadge status="approved" />);
    expect(container.querySelector(".text-emerald-300")).toBeInTheDocument();
  });

  it("applies red classes for rejected", () => {
    const { container } = render(<AIReviewBadge status="rejected" />);
    expect(container.querySelector(".text-red-300")).toBeInTheDocument();
  });
});

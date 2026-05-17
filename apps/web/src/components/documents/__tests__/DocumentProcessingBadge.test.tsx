import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { DocumentProcessingBadge } from "../DocumentProcessingBadge";

describe("DocumentProcessingBadge", () => {
  it.each([
    ["pending",    "Pending"],
    ["processing", "Processing"],
    ["completed",  "Processed"],
    ["failed",     "Failed"],
  ])("renders correct label for status %s", (status, label) => {
    render(<DocumentProcessingBadge status={status} />);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("falls back to pending for unknown status", () => {
    render(<DocumentProcessingBadge status="unknown_status" />);
    expect(screen.getByText("Pending")).toBeInTheDocument();
  });

  it("shows pulsing dot for processing status", () => {
    const { container } = render(<DocumentProcessingBadge status="processing" />);
    expect(container.querySelector(".animate-ping")).toBeInTheDocument();
  });

  it("does not show pulsing dot for non-processing statuses", () => {
    const { container } = render(<DocumentProcessingBadge status="completed" />);
    expect(container.querySelector(".animate-ping")).not.toBeInTheDocument();
  });
});

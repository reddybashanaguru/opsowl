import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ConfidenceBar } from "../ConfidenceBar";

describe("ConfidenceBar", () => {
  it("renders nothing when score is null", () => {
    const { container } = render(<ConfidenceBar score={null} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders nothing when score is undefined", () => {
    const { container } = render(<ConfidenceBar score={undefined} />);
    expect(container.firstChild).toBeNull();
  });

  it("shows high confidence label for score >= 0.8", () => {
    render(<ConfidenceBar score={0.85} />);
    expect(screen.getByText(/High/)).toBeInTheDocument();
    expect(screen.getByText(/85%/)).toBeInTheDocument();
  });

  it("shows medium confidence label for score 0.5–0.79", () => {
    render(<ConfidenceBar score={0.65} />);
    expect(screen.getByText(/Medium/)).toBeInTheDocument();
    expect(screen.getByText(/65%/)).toBeInTheDocument();
  });

  it("shows low confidence label for score < 0.5", () => {
    render(<ConfidenceBar score={0.3} />);
    expect(screen.getByText(/Low/)).toBeInTheDocument();
    expect(screen.getByText(/30%/)).toBeInTheDocument();
  });

  it("hides label when showLabel is false", () => {
    render(<ConfidenceBar score={0.9} showLabel={false} />);
    expect(screen.queryByText(/High/)).not.toBeInTheDocument();
  });

  it("renders progressbar with correct aria values", () => {
    render(<ConfidenceBar score={0.72} />);
    const bar = screen.getByRole("progressbar");
    expect(bar).toHaveAttribute("aria-valuenow", "72");
    expect(bar).toHaveAttribute("aria-valuemin", "0");
    expect(bar).toHaveAttribute("aria-valuemax", "100");
  });

  it("applies green color for high confidence", () => {
    const { container } = render(<ConfidenceBar score={0.9} />);
    expect(container.querySelector(".bg-emerald-500")).toBeInTheDocument();
  });

  it("applies amber color for medium confidence", () => {
    const { container } = render(<ConfidenceBar score={0.6} />);
    expect(container.querySelector(".bg-amber-500")).toBeInTheDocument();
  });

  it("applies red color for low confidence", () => {
    const { container } = render(<ConfidenceBar score={0.2} />);
    expect(container.querySelector(".bg-red-500")).toBeInTheDocument();
  });
});

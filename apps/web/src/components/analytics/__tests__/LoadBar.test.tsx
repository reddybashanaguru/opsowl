import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LoadBar } from "../LoadBar";

describe("LoadBar", () => {
  it("renders the fill element", () => {
    render(<LoadBar value={50} max={100} color="bg-cyan-500" />);
    expect(screen.getByTestId("loadbar-fill")).toBeInTheDocument();
  });

  it("sets width to 50% when value is half of max", () => {
    render(<LoadBar value={50} max={100} color="bg-cyan-500" />);
    const fill = screen.getByTestId("loadbar-fill");
    expect(fill).toHaveStyle({ width: "50%" });
  });

  it("sets width to 100% when value equals max", () => {
    render(<LoadBar value={10} max={10} color="bg-cyan-500" />);
    const fill = screen.getByTestId("loadbar-fill");
    expect(fill).toHaveStyle({ width: "100%" });
  });

  it("caps width at 100% when value exceeds max", () => {
    render(<LoadBar value={200} max={100} color="bg-cyan-500" />);
    const fill = screen.getByTestId("loadbar-fill");
    expect(fill).toHaveStyle({ width: "100%" });
  });

  it("sets width to 0% when max is 0", () => {
    render(<LoadBar value={5} max={0} color="bg-cyan-500" />);
    const fill = screen.getByTestId("loadbar-fill");
    expect(fill).toHaveStyle({ width: "0%" });
  });

  it("sets width to 0% when value is 0", () => {
    render(<LoadBar value={0} max={100} color="bg-cyan-500" />);
    const fill = screen.getByTestId("loadbar-fill");
    expect(fill).toHaveStyle({ width: "0%" });
  });

  it("applies the provided color class", () => {
    render(<LoadBar value={30} max={100} color="bg-violet-500" />);
    const fill = screen.getByTestId("loadbar-fill");
    expect(fill.className).toContain("bg-violet-500");
  });
});

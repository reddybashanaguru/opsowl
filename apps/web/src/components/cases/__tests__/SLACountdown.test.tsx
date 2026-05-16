import { render, screen, act } from "@testing-library/react";
import { SLACountdown } from "../SLACountdown";

describe("SLACountdown", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders 'No SLA' when breachAt is null", () => {
    render(<SLACountdown breachAt={null} breached={false} />);
    expect(screen.getByText("No SLA")).toBeInTheDocument();
  });

  it("renders 'SLA Breached' when breached=true", () => {
    const future = new Date(Date.now() + 3600000).toISOString();
    render(<SLACountdown breachAt={future} breached={true} />);
    expect(screen.getByText("SLA Breached")).toBeInTheDocument();
  });

  it("renders 'SLA Breached' when breachAt is in the past", () => {
    const past = new Date(Date.now() - 1000).toISOString();
    render(<SLACountdown breachAt={past} breached={false} />);
    // trigger effect tick
    act(() => { vi.advanceTimersByTime(0); });
    expect(screen.getByText("SLA Breached")).toBeInTheDocument();
  });

  it("renders countdown for future breach time", () => {
    const twoHoursOut = new Date(Date.now() + 2 * 3600000).toISOString();
    render(<SLACountdown breachAt={twoHoursOut} breached={false} />);
    act(() => { vi.advanceTimersByTime(0); });
    // Should show something like "2h 0m"
    expect(screen.getByText(/\dh/)).toBeInTheDocument();
  });

  it("shows amber warning color when under 2 hours remain", () => {
    const oneHourOut = new Date(Date.now() + 3600000).toISOString();
    const { container } = render(<SLACountdown breachAt={oneHourOut} breached={false} />);
    act(() => { vi.advanceTimersByTime(0); });
    const el = container.querySelector("span");
    expect(el).toHaveClass("text-amber-400");
  });

  it("shows normal color when more than 2 hours remain", () => {
    const threeHoursOut = new Date(Date.now() + 3 * 3600000).toISOString();
    const { container } = render(<SLACountdown breachAt={threeHoursOut} breached={false} />);
    act(() => { vi.advanceTimersByTime(0); });
    const el = container.querySelector("span");
    expect(el).toHaveClass("text-slate-400");
  });

  it("renders days when more than 24h remain", () => {
    const twoDaysOut = new Date(Date.now() + 2 * 24 * 3600000).toISOString();
    render(<SLACountdown breachAt={twoDaysOut} breached={false} />);
    act(() => { vi.advanceTimersByTime(0); });
    expect(screen.getByText(/\dd/)).toBeInTheDocument();
  });
});

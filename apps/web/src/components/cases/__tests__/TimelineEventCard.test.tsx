import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TimelineEventCard } from "../TimelineEventCard";
import type { TimelineEvent } from "@/lib/hooks/useTimelineSSE";

function makeEvent(overrides: Partial<TimelineEvent> = {}): TimelineEvent {
  return {
    id: "00000000-0000-0000-0000-000000000001",
    case_id: "00000000-0000-0000-0000-000000000002",
    event_type: "case_created",
    source: "user",
    actor: { id: "00000000-0000-0000-0000-000000000003", full_name: "Alice Smith", email: "alice@test.com" },
    occurred_at: new Date(Date.now() - 60_000).toISOString(), // 1 min ago
    created_at: new Date(Date.now() - 60_000).toISOString(),
    payload: {},
    back_dated: false,
    ...overrides,
  };
}

describe("TimelineEventCard", () => {
  it("renders event label and source", () => {
    render(<TimelineEventCard event={makeEvent()} />);
    expect(screen.getByText("Case Created")).toBeInTheDocument();
    expect(screen.getByText(/via user/)).toBeInTheDocument();
  });

  it("renders actor name", () => {
    render(<TimelineEventCard event={makeEvent()} />);
    expect(screen.getByText("Alice Smith")).toBeInTheDocument();
  });

  it("renders system event with no actor", () => {
    render(<TimelineEventCard event={makeEvent({ actor: null, source: "system" })} />);
    expect(screen.getByText("System")).toBeInTheDocument();
    expect(screen.getByText(/via system/)).toBeInTheDocument();
  });

  it("shows back_dated badge when flagged", () => {
    render(<TimelineEventCard event={makeEvent({ back_dated: true })} />);
    expect(screen.getByText("backdated")).toBeInTheDocument();
  });

  it("shows new badge when isNew=true", () => {
    render(<TimelineEventCard event={makeEvent()} isNew />);
    expect(screen.getByText("new")).toBeInTheDocument();
  });

  it("does not show new badge by default", () => {
    render(<TimelineEventCard event={makeEvent()} />);
    expect(screen.queryByText("new")).not.toBeInTheDocument();
  });

  it("expands payload on click", async () => {
    const user = userEvent.setup();
    const event = makeEvent({ payload: { foo: "bar" } });
    render(<TimelineEventCard event={event} />);
    // Payload not visible initially
    expect(screen.queryByText(/"foo"/)).not.toBeInTheDocument();
    // Click to expand
    await user.click(screen.getByText("Case Created"));
    expect(screen.getByText(/"foo"/)).toBeInTheDocument();
  });

  it("collapses payload on second click", async () => {
    const user = userEvent.setup();
    const event = makeEvent({ payload: { key: "value" } });
    render(<TimelineEventCard event={event} />);
    await user.click(screen.getByText("Case Created"));
    expect(screen.getByText(/"key"/)).toBeInTheDocument();
    await user.click(screen.getByText("Case Created"));
    expect(screen.queryByText(/"key"/)).not.toBeInTheDocument();
  });

  it("renders correct label for case_status_changed event", () => {
    render(<TimelineEventCard event={makeEvent({ event_type: "case_status_changed" })} />);
    expect(screen.getByText("Status Changed")).toBeInTheDocument();
  });

  it("renders correct label for note_added event", () => {
    render(<TimelineEventCard event={makeEvent({ event_type: "note_added" })} />);
    expect(screen.getByText("Note Added")).toBeInTheDocument();
  });

  it("renders unknown event_type with fallback label", () => {
    render(<TimelineEventCard event={makeEvent({ event_type: "some_unknown_type" })} />);
    expect(screen.getByText("some_unknown_type")).toBeInTheDocument();
  });

  it("shows relative time", () => {
    render(<TimelineEventCard event={makeEvent()} />);
    // Should show something like "1m ago"
    expect(screen.getByText(/ago/)).toBeInTheDocument();
  });

  it("shows absolute timestamp in title attribute", () => {
    const event = makeEvent();
    render(<TimelineEventCard event={event} />);
    const timeEl = screen.getByText(/ago/);
    expect(timeEl).toHaveAttribute("title");
  });
});
